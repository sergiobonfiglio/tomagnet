package search

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/tidwall/gjson"
	"golang.org/x/net/html"
	"golang.org/x/text/encoding/htmlindex"
	"github.com/sergiobonfiglio/tomagnet/internal/cardigann"
	"github.com/sergiobonfiglio/tomagnet/internal/config"
	"github.com/sergiobonfiglio/tomagnet/internal/definitions"
	"github.com/sergiobonfiglio/tomagnet/internal/fetch"
	"github.com/sergiobonfiglio/tomagnet/internal/normalize"
)

type Options struct {
	Query              string
	Mode               string
	Season             string
	Episode            string
	IMDBID             string
	TMDBID             string
	TVDBID             string
	DoubanID           string
	TVMazeID           string
	Artist             string
	Album              string
	Author             string
	Title              string
	Genre              string
	Year               string
	Categories         []string
	Indexers           []config.Indexer
	Limit, Concurrency int
	Debug              func(string, ...any)
}

func (o Options) cardigann() cardigann.SearchOptions {
	cats := o.Categories
	if len(cats) == 0 {
		cats = nil
	}
	return cardigann.SearchOptions{
		Keywords:   o.Query,
		Categories: cats,
		Mode:       o.Mode,
		Season:     o.Season,
		Episode:    o.Episode,
		IMDBID:     o.IMDBID,
		TMDBID:     o.TMDBID,
		TVDBID:     o.TVDBID,
		DoubanID:   o.DoubanID,
		TVMazeID:   o.TVMazeID,
		Artist:     o.Artist,
		Album:      o.Album,
		Author:     o.Author,
		Title:      o.Title,
		Genre:      o.Genre,
		Year:       o.Year,
	}
}

type requester struct {
	delay   time.Duration
	timeout time.Duration
	debug   func(string, ...any)
	last    time.Time
	now     func() time.Time
	sleep   func(context.Context, time.Duration) error
	do      func(context.Context, fetch.Request, time.Duration, func(string, ...any)) (fetch.Response, error)
}

func newRequester(d *cardigann.Definition, timeout time.Duration, debug func(string, ...any)) requester {
	session := fetch.NewSession(timeout, debug, cardigann.Certificates(d)...)
	return requester{
		delay:   requestDelay(d),
		timeout: timeout,
		debug:   debug,
		now:     time.Now,
		sleep: func(ctx context.Context, d time.Duration) error {
			t := time.NewTimer(d)
			defer t.Stop()
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-t.C:
				return nil
			}
		},
		do: func(ctx context.Context, req fetch.Request, timeout time.Duration, debug func(string, ...any)) (fetch.Response, error) {
			return session.Do(ctx, req)
		},
	}
}

func requestDelay(d *cardigann.Definition) time.Duration {
	if d == nil || d.Raw == nil {
		return 0
	}
	f, err := strconv.ParseFloat(strings.TrimSpace(fmt.Sprint(d.Raw["requestDelay"])), 64)
	if err != nil || f <= 0 {
		return 0
	}
	return time.Duration(f * float64(time.Second))
}

func queryValue(d *cardigann.Definition, opt Options) string {
	cg := opt.cardigann()
	for _, name := range cardigann.ModeParamNames(d, firstNonEmpty(opt.Mode, "search")) {
		switch strings.ToLower(name) {
		case "q":
			if cg.Keywords != "" {
				return cg.Keywords
			}
		case "season":
			if cg.Season != "" {
				return cg.Season
			}
		case "ep", "episode":
			if cg.Episode != "" {
				return cg.Episode
			}
		case "imdbid":
			if cg.IMDBID != "" {
				return cg.IMDBID
			}
		case "tmdbid":
			if cg.TMDBID != "" {
				return cg.TMDBID
			}
		case "tvdbid":
			if cg.TVDBID != "" {
				return cg.TVDBID
			}
		case "doubanid":
			if cg.DoubanID != "" {
				return cg.DoubanID
			}
		case "tvmazeid":
			if cg.TVMazeID != "" {
				return cg.TVMazeID
			}
		case "artist":
			if cg.Artist != "" {
				return cg.Artist
			}
		case "album":
			if cg.Album != "" {
				return cg.Album
			}
		case "author":
			if cg.Author != "" {
				return cg.Author
			}
		case "title":
			if cg.Title != "" {
				return cg.Title
			}
		case "genre":
			if cg.Genre != "" {
				return cg.Genre
			}
		case "year":
			if cg.Year != "" {
				return cg.Year
			}
		}
	}
	return opt.Query
}

func firstNonEmpty(vs ...string) string {
	for _, v := range vs {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func (r *requester) Do(ctx context.Context, req fetch.Request) (fetch.Response, error) {
	if r.now == nil {
		r.now = time.Now
	}
	if r.sleep == nil {
		r.sleep = func(context.Context, time.Duration) error { return nil }
	}
	if r.do == nil {
		r.do = fetch.Do
	}
	if r.delay > 0 && !r.last.IsZero() {
		if wait := r.delay - r.now().Sub(r.last); wait > 0 {
			if err := r.sleep(ctx, wait); err != nil {
				return fetch.Response{}, err
			}
		}
	}
	r.last = r.now()
	return r.do(ctx, req, r.timeout, r.debug)
}

func Run(ctx context.Context, opt Options) Response {
	start := time.Now().UTC()
	if opt.Concurrency <= 0 {
		opt.Concurrency = 4
	}
	if opt.Debug == nil {
		opt.Debug = func(string, ...any) {}
	}
	resp := Response{Results: []Result{}, Errors: []Error{}, Meta: Meta{Query: opt.Query, StartedAt: start, IndexersRequested: len(opt.Indexers)}}
	sem := make(chan struct{}, opt.Concurrency)
	var mu sync.Mutex
	var wg sync.WaitGroup
	for _, idx := range opt.Indexers {
		idx := idx
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			rs, err := runOne(ctx, idx, opt)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				resp.Errors = append(resp.Errors, Error{Indexer: idx.ID, Stage: "search", Message: err.Error()})
				resp.Meta.IndexersFailed++
			} else {
				resp.Results = append(resp.Results, rs...)
				resp.Meta.IndexersSucceeded++
			}
		}()
	}
	wg.Wait()
	resp.Meta.TotalResults = len(resp.Results)
	resp.Meta.DurationMS = time.Since(start).Milliseconds()
	return resp
}

func runOne(ctx context.Context, idx config.Indexer, opt Options) ([]Result, error) {
	p, err := definitions.Resolve(idx.ID)
	if err != nil {
		return nil, fmt.Errorf("resolve: %w", err)
	}
	opt.Debug("definition %s %s", idx.ID, p)
	d, err := cardigann.Load(p)
	if err != nil {
		return nil, fmt.Errorf("parse definition: %w", err)
	}
	if err := d.Validate(); err != nil {
		return nil, fmt.Errorf("validate: %w", err)
	}
	bases, auto := baseURLs(idx, d)
	rs, err := tryBases(bases, func(base string) ([]Result, error) {
		dd := *d
		dd.BaseURL = base
		rs, err := runOneBase(ctx, &dd, idx, opt)
		if err == nil && auto {
			cacheBaseURL(idx.ID, base, opt.Debug)
		}
		return rs, err
	})
	if err != nil && auto {
		return nil, fmt.Errorf("%w; consider disabling indexer %q via disabled_indexers", err, idx.ID)
	}
	return rs, err
}

func baseURLs(idx config.Indexer, d *cardigann.Definition) ([]string, bool) {
	if !strings.EqualFold(strings.TrimSpace(idx.BaseURL), "auto") {
		return cardigann.BaseURLs(d, idx.BaseURL), false
	}
	bases := cardigann.BaseURLs(d, "")
	if cached := cachedBaseURL(idx.ID); cached != "" {
		bases = append([]string{cached}, bases...)
	}
	return uniqueStrings(bases), true
}

func cachedBaseURL(id string) string {
	b, err := os.ReadFile(baseURLCachePath(id))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

func cacheBaseURL(id, base string, debug func(string, ...any)) {
	path := baseURLCachePath(id)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		debug("cache base url mkdir %s: %v", id, err)
		return
	}
	if err := os.WriteFile(path, []byte(strings.TrimSpace(base)+"\n"), 0o644); err != nil {
		debug("cache base url write %s: %v", id, err)
	}
}

func baseURLCachePath(id string) string {
	dir, err := os.UserCacheDir()
	if err != nil || dir == "" {
		dir = filepath.Join(os.TempDir(), "tomagnet-cache")
	}
	return filepath.Join(dir, "tomagnet", "base_urls", id)
}

func uniqueStrings(in []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}

func tryBases[T any](bases []string, fn func(string) (T, error)) (T, error) {
	var zero T
	if len(bases) == 0 {
		return zero, fmt.Errorf("no base urls")
	}
	var last error
	for _, base := range bases {
		v, err := fn(base)
		if err == nil {
			return v, nil
		}
		last = err
	}
	if last == nil {
		last = fmt.Errorf("all base urls failed")
	}
	return zero, fmt.Errorf("all base urls failed: %w", last)
}

func runOneBase(ctx context.Context, d *cardigann.Definition, idx config.Indexer, opt Options) ([]Result, error) {
	rq := newRequester(d, time.Duration(idx.TimeoutSeconds)*time.Second, opt.Debug)
	loginCookies := map[string]string{}
	login := cardigann.LoginRequest(d)
	if err := loginUnsupported(d); err != nil && login.Path != "" && !strings.EqualFold(login.Method, "cookie") {
		return nil, err
	}
	if strings.EqualFold(login.Method, "cookie") {
		if c := loginCookieHeader(login); c != "" {
			loginCookies["__raw__"] = c
		}
	} else if login.Path != "" {
		req := login
		if loginNeedsPage(d, login) {
			pre, err := rq.Do(ctx, fetch.Request{Method: "get", Base: d.BaseURL, Path: login.Path, Headers: login.Headers, FollowRedirect: &login.FollowRedirect})
			if err != nil {
				return nil, fmt.Errorf("login page: %w", err)
			}
			loginCookies = mergeCookies(loginCookies, pre.Cookies)
			req = buildLoginRequest(d, string(pre.Body))
			if req.Headers == nil {
				req.Headers = map[string]string{}
			}
			if req.Headers["Cookie"] == "" && len(loginCookies) > 0 {
				req.Headers["Cookie"] = cookieHeader(loginCookies)
			}
		}
		lr, err := rq.Do(ctx, fetch.Request{Method: req.Method, Base: d.BaseURL, Path: req.Path, Inputs: req.Inputs, Headers: req.Headers, FollowRedirect: &req.FollowRedirect})
		if err != nil {
			return nil, fmt.Errorf("login: %w", err)
		}
		loginCookies = mergeCookies(loginCookies, lr.Cookies)
	}
	if err := verifyLogin(ctx, d, loginCookies, rq.Do); err != nil {
		return nil, err
	}
	spec := cardigann.SearchRequestWithOptions(d, opt.cardigann())
	if spec.Headers["Cookie"] == "" {
		if raw := loginCookies["__raw__"]; raw != "" {
			spec.Headers["Cookie"] = raw
		} else if len(loginCookies) > 0 {
			spec.Headers["Cookie"] = cookieHeader(loginCookies)
		}
	}
	param := cardigann.QueryParamForMode(d, firstNonEmpty(opt.Mode, "search"))
	if strings.Contains(spec.Path, "?") || len(spec.Inputs) > 0 {
		param = ""
	}
	fr, err := rq.Do(ctx, fetch.Request{Method: spec.Method, Base: d.BaseURL, Path: spec.Path, Param: param, Query: queryValue(d, opt), Inputs: spec.Inputs, Headers: spec.Headers, FollowRedirect: &spec.FollowRedirect})
	if err != nil {
		return nil, fmt.Errorf("request: %w", err)
	}
	rs, err := Parse(idx.ID, d, fr.Body, fr.ContentType, opt.Limit)
	if err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}
	rs = EnrichDetails(ctx, d, rs, func(ctx context.Context, req fetch.Request) ([]byte, string, error) {
		if req.FollowRedirect == nil {
			fr := cardigann.FollowRedirect(d)
			req.FollowRedirect = &fr
		}
		fr, err := rq.Do(ctx, req)
		return fr.Body, fr.ContentType, err
	})
	return rs, nil
}

func Parse(indexer string, d *cardigann.Definition, b []byte, ct string, limit int) ([]Result, error) {
	s := cardigann.Preprocess(d, decodeBody(d, b))
	if err := checkErrorSelectors(d, s); err != nil {
		return nil, err
	}
	if strings.Contains(ct, "json") || cardigann.ResponseType(d) == "json" || gjson.Valid(s) {
		return parseJSON(indexer, d, s, limit), nil
	}
	return parseHTML(indexer, d, s, limit)
}

func decodeBody(d *cardigann.Definition, b []byte) string {
	encName := strings.TrimSpace(fmt.Sprint(d.Raw["encoding"]))
	if encName == "" || strings.EqualFold(encName, "utf-8") {
		return string(b)
	}
	enc, err := htmlindex.Get(encName)
	if err != nil {
		return string(b)
	}
	r := enc.NewDecoder().Reader(bytes.NewReader(b))
	decoded, err := io.ReadAll(r)
	if err != nil {
		return string(b)
	}
	return string(decoded)
}

func checkErrorSelectors(d *cardigann.Definition, s string) error {
	if gjson.Valid(s) {
		return nil
	}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(s))
	if err != nil {
		return nil
	}
	for _, ev := range cardigann.ErrorSelectors(d) {
		found := selectNodes(doc.Selection, ev.Selector)
		if found.Length() == 0 {
			continue
		}
		msg := found.First().Text()
		if ev.MessageSelector != "" {
			msg = selectNodes(doc.Selection, ev.MessageSelector).First().Text()
		}
		if strings.TrimSpace(msg) == "" {
			msg = "indexer returned error"
		}
		return fmt.Errorf("indexer error: %s", strings.TrimSpace(msg))
	}
	return nil
}

func parseJSON(indexer string, d *cardigann.Definition, s string, limit int) []Result {
	rowPath := cardigann.ResultsSelector(d)
	arr := gjson.Result{}
	if rowPath != "" && rowPath != "item, entry, tr" {
		arr = jsonGet(s, rowPath)
	}
	if !arr.Exists() {
		arr = gjson.Get(s, "results")
	}
	if !arr.Exists() {
		arr = gjson.Get(s, "data")
	}
	if !arr.Exists() {
		arr = gjson.Parse(s)
	}
	if countSel := cardigann.RowsCountSelector(d); countSel != "" {
		if countIsZero(jsonGet(s, countSel)) {
			return nil
		}
	}
	rows := jsonRows(arr, cardigann.RowsAttribute(d), cardigann.RowsMissingAttributeEqualsNoResults(d))
	out := []Result{}
	for _, row := range rows {
		vars := map[string]string{}
		get := func(field string, fallbacks ...string) string {
			val := ""
			if t := cardigann.FieldText(d, field, vars); t != "" {
				val = t
			} else if p := cardigann.FieldSelector(d, field); p != "" {
				if strings.HasPrefix(p, "..") {
					if x := row.parent.Get(strings.TrimPrefix(p, "..")); x.Exists() {
						val = x.String()
					}
				} else if x := row.item.Get(p); x.Exists() {
					val = x.String()
				}
			}
			if val == "" {
				for _, p := range fallbacks {
					if x := row.item.Get(p); x.Exists() {
						val = x.String()
						break
					}
				}
			}
			if val == "" {
				val = cardigann.FieldDefault(d, field, vars)
			}
			val = cardigann.ApplyFilters(d, field, val, vars)
			return cardigann.FieldCase(d, field, val)
		}
		for pass := 0; pass < 5; pass++ {
			changed := false
			for _, f := range cardigann.FieldNames(d) {
				v := get(f, f)
				if vars[f] != v {
					vars[f] = v
					changed = true
				}
			}
			if !changed {
				break
			}
		}
		r := Result{Indexer: indexer}
		r.Title = normalize.StrPtr(first(vars["title"], get("title", "name")))
		r.GUID = normalize.StrPtr(first(vars["guid"], get("guid", "id")))
		ih := strings.ToUpper(first(vars["infohash"], get("infohash", "info_hash")))
		r.InfoHash = normalize.StrPtr(ih)
		r.MagnetURL = normalize.StrPtr(first(vars["magnet"], get("magnet", "magnet_url")))
		if r.MagnetURL == nil && ih != "" {
			mag := "magnet:?xt=urn:btih:" + ih
			r.MagnetURL = &mag
		}
		r.DownloadURL = normalize.StrPtr(normalize.Abs(d.BaseURL, first(vars["download"], get("download", "download_url", "link"))))
		r.DetailsURL = normalize.StrPtr(normalize.Abs(d.BaseURL, first(vars["details"], get("details", "details_url", "comments"))))
		r.Size = normalize.Size(first(vars["size"], get("size", "bytes")))
		r.Seeders = normalize.Int(first(vars["seeders"], get("seeders", "seeds")))
		r.Leechers = normalize.Int(first(vars["leechers"], get("leechers", "peers")))
		r.PublishDate = normalize.Date(first(vars["date"], get("date", "publish_date", "pubDate")))
		r.Category = normalize.StrPtr(first(vars["category"], get("category")))
		out = append(out, r)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out
}

type jsonRow struct {
	parent gjson.Result
	item   gjson.Result
}

func jsonRows(arr gjson.Result, attr string, missingNoResults bool) []jsonRow {
	if attr == "" {
		rows := []jsonRow{}
		arr.ForEach(func(_, v gjson.Result) bool { rows = append(rows, jsonRow{item: v}); return true })
		return rows
	}
	if arr.IsArray() {
		rows := []jsonRow{}
		arr.ForEach(func(_, parent gjson.Result) bool {
			if child := parent.Get(attr); child.Exists() {
				child.ForEach(func(_, v gjson.Result) bool { rows = append(rows, jsonRow{parent: parent, item: v}); return true })
			} else if !missingNoResults {
				rows = append(rows, jsonRow{item: parent})
			}
			return true
		})
		return rows
	}
	if child := arr.Get(attr); child.Exists() {
		rows := []jsonRow{}
		child.ForEach(func(_, v gjson.Result) bool { rows = append(rows, jsonRow{parent: arr, item: v}); return true })
		return rows
	}
	if missingNoResults {
		return nil
	}
	return []jsonRow{{item: arr}}
}

func jsonGet(s, path string) gjson.Result {
	path = strings.TrimSpace(path)
	if path == "" || path == "$" {
		return gjson.Parse(s)
	}
	path = strings.TrimPrefix(path, "$")
	path = strings.TrimPrefix(path, ".")
	path = strings.ReplaceAll(path, "[", ".")
	path = strings.ReplaceAll(path, "]", "")
	path = strings.TrimPrefix(path, ".")
	if path == "" {
		return gjson.Parse(s)
	}
	return gjson.Get(s, path)
}

func countIsZero(v gjson.Result) bool {
	if !v.Exists() {
		return false
	}
	if v.IsArray() {
		return len(v.Array()) == 0
	}
	s := strings.TrimSpace(v.String())
	if s == "" || s == "0" || s == "false" || s == "null" {
		return true
	}
	if v.Type == gjson.True || v.Type == gjson.False {
		return !v.Bool()
	}
	return false
}

func parseHTML(indexer string, d *cardigann.Definition, s string, limit int) ([]Result, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(s))
	if err != nil {
		return nil, err
	}
	sel := cardigann.ResultsSelector(d)
	removeSel := cardigann.RowsRemove(d)
	dateHeaderSel := cardigann.RowsDateHeaderSelector(d)
	after := cardigann.RowsAfter(d)
	out := []Result{}
	selectNodes(doc.Selection, sel).EachWithBreak(func(i int, row *goquery.Selection) bool {
		if after > 0 {
			nodes := []*html.Node{}
			for p, n := row, 0; n < after; n++ {
				p = p.Next()
				if p.Length() == 0 || len(p.Nodes) == 0 {
					break
				}
				nodes = append(nodes, p.Nodes[0])
			}
			if len(nodes) > 0 {
				row = row.AddNodes(nodes...)
			}
		}
		if removeSel != "" {
			selectNodes(row, removeSel).Remove()
		}
		if filters := cardigann.RowFilters(d); len(filters) > 0 {
			h, _ := goquery.OuterHtml(row)
			h = cardigann.ApplyFilterList(d, filters, h, nil)
			rd, err := goquery.NewDocumentFromReader(strings.NewReader(h))
			if err == nil {
				row = rd.Selection.Children().First()
			}
		}
		dateHeader := ""
		if dateHeaderSel != "" {
			for p := row.Prev(); p.Length() > 0; p = p.Prev() {
				if matchesSelector(p, dateHeaderSel) {
					dateHeader = strings.TrimSpace(p.Text())
					break
				}
			}
		}
		r := Result{Indexer: indexer}
		vars := map[string]string{}
		text := func(f string) string {
			val := ""
			if t := cardigann.FieldText(d, f, vars); t != "" {
				val = t
			} else if fs := cardigann.FieldSelector(d, f); fs != "" {
				q := selectNodes(row, fs)
				if rm := cardigann.FieldRemove(d, f); rm != "" {
					q.Find(cardigann.CSSSelector(rm)).Remove()
				}
				if attr := cardigann.FieldAttr(d, f); attr != "" {
					val, _ = q.Attr(attr)
				} else {
					val = q.Text()
				}
			}
			if val == "" {
				val = cardigann.FieldDefault(d, f, vars)
			}
			val = cardigann.ApplyFilters(d, f, val, vars)
			val = cardigann.FieldCase(d, f, val)
			vars[f] = val
			return val
		}
		r.Title = normalize.StrPtr(first(text("title"), selectNodes(row, "a").First().Text()))
		r.MagnetURL = normalize.StrPtr(first(text("magnet"), attr(row, "a[href^='magnet:']", "href")))
		r.DownloadURL = normalize.StrPtr(normalize.Abs(d.BaseURL, first(text("download"), attr(row, "a[href$='.torrent']", "href"))))
		r.DetailsURL = normalize.StrPtr(normalize.Abs(d.BaseURL, first(text("details"), attr(row, "a", "href"))))
		r.Size = normalize.Size(text("size"))
		r.Seeders = normalize.Int(text("seeders"))
		r.Leechers = normalize.Int(text("leechers"))
		r.PublishDate = normalize.Date(first(text("date"), dateHeader))
		r.Category = normalize.StrPtr(text("category"))
		out = append(out, r)
		return limit <= 0 || len(out) < limit
	})
	return out, nil
}
func cookieHeader(cookies map[string]string) string {
	if len(cookies) == 0 {
		return ""
	}
	keys := make([]string, 0, len(cookies))
	for k := range cookies {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, k+"="+cookies[k])
	}
	return strings.Join(parts, "; ")
}

func attr(s *goquery.Selection, sel, a string) string { v, _ := s.Find(sel).First().Attr(a); return v }
func first(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return a
	}
	return b
}

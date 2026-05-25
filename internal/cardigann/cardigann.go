package cardigann

import (
	"encoding/json"
	"fmt"
	"html"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Definition struct {
	ID, Name, BaseURL string
	Config            map[string]string
	Raw               map[string]any
}

type RequestSpec struct {
	Method         string
	Path           string
	Inputs         map[string]string
	Headers        map[string]string
	FollowRedirect bool
}

type SearchOptions struct {
	Keywords   string
	Categories []string
	Mode       string
	Season     string
	Episode    string
	IMDBID     string
	TMDBID     string
	TVDBID     string
	DoubanID   string
	TVMazeID   string
	Artist     string
	Album      string
	Author     string
	Title      string
	Genre      string
	Year       string
}

func Load(path string) (*Definition, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var raw map[string]any
	if err := yaml.Unmarshal(b, &raw); err != nil {
		fixed := []byte(strings.ReplaceAll(string(b), `\/`, `/`))
		if err2 := yaml.Unmarshal(fixed, &raw); err2 != nil {
			return nil, err
		}
	}
	d := FromRaw(raw)
	if d.ID == "" {
		d.ID = strings.TrimSuffix(strings.TrimSuffix(filepath.Base(path), ".yaml"), ".yml")
	}
	return d, nil
}

func FromRaw(raw map[string]any) *Definition {
	d := &Definition{Raw: raw, Config: settingDefaults(raw)}
	d.ID = str(raw["id"])
	d.Name = str(raw["name"])
	d.BaseURL = firstStr(raw, "base_url", "site", "url")
	if d.BaseURL == "" {
		if links := slice(raw["links"]); len(links) > 0 {
			d.BaseURL = str(links[0])
		}
	}
	if d.Config["sitelink"] == "" {
		d.Config["sitelink"] = d.BaseURL
	}
	return d
}

func (d *Definition) Validate() error {
	if d.BaseURL == "" && SearchPath(d, "") == "" {
		return fmt.Errorf("missing base_url/site/url/links")
	}
	return nil
}

func SearchPath(d *Definition, query string) string {
	return SearchRequest(d, query).Path
}

func SearchRequest(d *Definition, keywords string) RequestSpec {
	return SearchRequestWithOptions(d, SearchOptions{Keywords: keywords, Categories: DefaultCategories(d)})
}

func SearchRequestWithOptions(d *Definition, opt SearchOptions) RequestSpec {
	search := mapAny(d.Raw["search"])
	kw := ApplyFilterList(d, slice(search["keywordsfilters"]), opt.Keywords, nil)
	q := buildQueryMap(opt, kw)
	pathMap := map[string]any{}
	path := str(search["path"])
	if paths := slice(search["paths"]); len(paths) > 0 {
		pathMap = selectPath(paths, opt.Categories)
		if str(pathMap["path"]) != "" {
			path = str(pathMap["path"])
		}
	}
	if path == "" {
		path = "/"
	}
	method := renderedMethod(first(anyString(pathMap["method"]), anyString(search["method"]), "get"), d.Config, q, nil)
	inputs := map[string]string{}
	if inherit, ok := pathMap["inheritinputs"]; !ok || fmt.Sprint(inherit) != "false" {
		mergeRendered(inputs, mapAny(search["inputs"]), d.Config, q, nil)
	}
	mergeRendered(inputs, mapAny(pathMap["inputs"]), d.Config, q, nil)
	headers := map[string]string{}
	for k, v := range mapAny(search["headers"]) {
		if ss := slice(v); len(ss) > 0 {
			headers[k] = Render(anyString(ss[0]), d.Config, q, nil)
		} else {
			headers[k] = Render(anyString(v), d.Config, q, nil)
		}
	}
	if cookies := renderedCookies(mapAny(search["cookies"]), d.Config, q); cookies != "" {
		headers["Cookie"] = cookies
	}
	return RequestSpec{Method: method, Path: Render(path, d.Config, q, nil), Inputs: inputs, Headers: headers, FollowRedirect: searchFollowRedirect(d, pathMap)}
}

func buildQueryMap(opt SearchOptions, keywords string) map[string]string {
	q := map[string]string{
		"Keywords":    keywords,
		"Query":       url.QueryEscape(keywords),
		"Q":           url.QueryEscape(keywords),
		"Categories":  strings.Join(opt.Categories, ","),
		"Season":      opt.Season,
		"Ep":          opt.Episode,
		"Episode":     opt.Episode,
		"IMDBID":      opt.IMDBID,
		"IMDBIDShort": strings.TrimPrefix(strings.ToLower(opt.IMDBID), "tt"),
		"TMDBID":      opt.TMDBID,
		"TVDBID":      opt.TVDBID,
		"DoubanID":    opt.DoubanID,
		"TVMazeID":    opt.TVMazeID,
		"Artist":      opt.Artist,
		"Album":       opt.Album,
		"Author":      opt.Author,
		"Title":       opt.Title,
		"Genre":       opt.Genre,
		"Year":        opt.Year,
		"Type":        opt.Mode,
		"Mode":        opt.Mode,
	}
	for _, k := range []string{"Season", "Ep", "Episode", "IMDBID", "TMDBID", "TVDBID", "DoubanID", "TVMazeID", "Artist", "Album", "Author", "Title", "Genre", "Year", "Type", "Mode"} {
		if q[k] == "" {
			q[k] = "false"
		}
	}
	return q
}

func mergeRendered(dst map[string]string, src map[string]any, cfg, query, result map[string]string) {
	for k, v := range src {
		dst[k] = Render(anyString(v), cfg, query, result)
	}
}

func LoginRequest(d *Definition) RequestSpec {
	login := mapAny(d.Raw["login"])
	if len(login) == 0 {
		return RequestSpec{}
	}
	q := map[string]string{}
	inputs := map[string]string{}
	mergeRendered(inputs, mapAny(login["inputs"]), d.Config, q, nil)
	headers := map[string]string{}
	for k, v := range mapAny(login["headers"]) {
		if ss := slice(v); len(ss) > 0 {
			headers[k] = Render(anyString(ss[0]), d.Config, q, nil)
		} else {
			headers[k] = Render(anyString(v), d.Config, q, nil)
		}
	}
	if cookies := renderedCookies(mapAny(login["cookies"]), d.Config, q); cookies != "" {
		headers["Cookie"] = cookies
	}
	return RequestSpec{Method: renderedMethod(first(anyString(login["method"]), "get"), d.Config, q, nil), Path: Render(anyString(login["path"]), d.Config, q, nil), Inputs: inputs, Headers: headers, FollowRedirect: FollowRedirect(d)}
}

func DownloadBeforeRequest(d *Definition, rawURL string) RequestSpec {
	before := mapAny(dotted(d.Raw, "download.before"))
	if len(before) == 0 {
		return RequestSpec{}
	}
	q := downloadTemplateVars(rawURL)
	inputs := map[string]string{}
	for k, v := range mapAny(before["inputs"]) {
		inputs[k] = Render(anyString(v), d.Config, q, nil)
	}
	headers := map[string]string{}
	for k, v := range mapAny(before["headers"]) {
		headers[k] = Render(anyString(v), d.Config, q, nil)
	}
	return RequestSpec{
		Method:         renderedMethod(first(anyString(before["method"]), "get"), d.Config, q, nil),
		Path:           Render(anyString(before["path"]), d.Config, q, nil),
		Inputs:         inputs,
		Headers:        headers,
		FollowRedirect: FollowRedirect(d),
	}
}

func downloadTemplateVars(rawURL string) map[string]string {
	out := map[string]string{}
	u, err := url.Parse(rawURL)
	if err != nil {
		return out
	}
	out["DownloadUri.AbsoluteUri"] = u.String()
	out["DownloadUri.AbsolutePath"] = u.Path
	out["DownloadUri.PathAndQuery"] = u.RequestURI()
	for k, vs := range u.Query() {
		if len(vs) > 0 {
			out["DownloadUri.Query."+k] = vs[0]
		}
	}
	return out
}

func renderedCookies(src map[string]any, cfg, query map[string]string) string {
	if len(src) == 0 {
		return ""
	}
	keys := make([]string, 0, len(src))
	for k := range src {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, k+"="+Render(anyString(src[k]), cfg, query, nil))
	}
	return strings.Join(parts, "; ")
}

func FollowRedirect(d *Definition) bool {
	if d == nil || d.Raw == nil {
		return false
	}
	return fmt.Sprint(d.Raw["followredirect"]) == "true"
}

func Certificates(d *Definition) []string {
	out := []string{}
	for _, v := range slice(d.Raw["certificates"]) {
		if s := strings.TrimSpace(anyString(v)); s != "" {
			out = append(out, s)
		}
	}
	return out
}

func BaseURLs(d *Definition, override string) []string {
	if s := strings.TrimSpace(override); s != "" {
		return []string{s}
	}
	seen := map[string]bool{}
	out := []string{}
	add := func(s string) {
		s = strings.TrimSpace(s)
		if s == "" || seen[s] {
			return
		}
		seen[s] = true
		out = append(out, s)
	}
	add(d.BaseURL)
	for _, v := range slice(d.Raw["links"]) {
		add(anyString(v))
	}
	for _, v := range slice(d.Raw["legacylinks"]) {
		add(anyString(v))
	}
	return out
}

func searchFollowRedirect(d *Definition, pathMap map[string]any) bool {
	if v, ok := pathMap["followredirect"]; ok {
		return fmt.Sprint(v) == "true"
	}
	return FollowRedirect(d)
}

func selectPath(paths []any, cats []string) map[string]any {
	if len(cats) > 0 {
		want := map[string]bool{}
		for _, c := range cats {
			want[c] = true
		}
		for _, p := range paths {
			m := mapAny(p)
			for _, c := range slice(m["categories"]) {
				if want[anyString(c)] {
					return m
				}
			}
		}
	}
	return mapAny(paths[0])
}

func DefaultCategories(d *Definition) []string {
	var out []string
	for _, v := range slice(dotted(d.Raw, "caps.categorymappings")) {
		m := mapAny(v)
		if fmt.Sprint(m["default"]) == "true" {
			out = append(out, anyString(firstAny(m["cat"], m["id"])))
		}
	}
	return out
}

func ResponseType(d *Definition) string {
	search := mapAny(d.Raw["search"])
	if paths := slice(search["paths"]); len(paths) > 0 {
		if m := mapAny(paths[0]); len(m) > 0 {
			if r := mapAny(m["response"]); str(r["type"]) != "" {
				return str(r["type"])
			}
		}
	}
	return ""
}

func QueryParam(d *Definition) string { return QueryParamForMode(d, "search") }

func ModeParamNames(d *Definition, mode string) []string {
	caps := mapAny(d.Raw["caps"])
	modes := mapAny(caps["modes"])
	m := modes[mode]
	if arr := slice(m); len(arr) > 0 {
		out := make([]string, 0, len(arr))
		for _, v := range arr {
			if s := str(v); s != "" {
				out = append(out, s)
			}
		}
		if len(out) > 0 {
			return out
		}
	}
	search := mapAny(m)
	if ps := slice(search["params"]); len(ps) > 0 {
		out := make([]string, 0, len(ps))
		for _, v := range ps {
			if mm := mapAny(v); str(mm["name"]) != "" {
				out = append(out, str(mm["name"]))
			}
		}
		if len(out) > 0 {
			return out
		}
	}
	if mode != "search" {
		return ModeParamNames(d, "search")
	}
	return []string{"q"}
}

func QueryParamForMode(d *Definition, mode string) string {
	ps := ModeParamNames(d, mode)
	if len(ps) > 0 && ps[0] != "" {
		return ps[0]
	}
	return "q"
}

func AllowEmptyInputs(d *Definition) bool {
	return fmt.Sprint(dotted(d.Raw, "search.allowEmptyInputs")) == "true"
}

func ResultsSelector(d *Definition) string {
	if s := str(dotted(d.Raw, "search.rows.selector")); s != "" {
		return Render(s, d.Config, nil, nil)
	}
	return "item, entry, tr"
}

func FieldSelector(d *Definition, field string) string {
	for _, p := range []string{"search.fields." + field + ".selector", "fields." + field + ".selector", "search." + field + ".selector"} {
		if s := str(dotted(d.Raw, p)); s != "" {
			return s
		}
	}
	return ""
}

func CSSSelector(sel string) string {
	if !strings.HasPrefix(sel, "/") && !strings.HasPrefix(sel, "./") {
		return sel
	}
	s := strings.TrimPrefix(sel, "//")
	s = strings.TrimPrefix(s, "./")
	s = strings.TrimPrefix(s, "/")
	parts := strings.Split(s, "/")
	re := regexp.MustCompile(`^([A-Za-z0-9_-]+)\[@([^=]+)='([^']*)'\]$`)
	for i, p := range parts {
		p = strings.TrimSpace(p)
		if m := re.FindStringSubmatch(p); len(m) == 4 {
			p = m[1] + `[` + m[2] + `="` + m[3] + `"]`
		}
		parts[i] = p
	}
	return strings.Join(parts, " ")
}

func FieldText(d *Definition, field string, result map[string]string) string {
	for _, p := range []string{"search.fields." + field + ".text", "fields." + field + ".text"} {
		if s := str(dotted(d.Raw, p)); s != "" {
			return Render(s, d.Config, nil, result)
		}
	}
	return ""
}

func FieldAttr(d *Definition, field string) string {
	for _, p := range []string{"search.fields." + field + ".attribute", "fields." + field + ".attribute"} {
		if s := str(dotted(d.Raw, p)); s != "" {
			return s
		}
	}
	return ""
}

func FieldNames(d *Definition) []string {
	fields := mapAny(dotted(d.Raw, "search.fields"))
	out := make([]string, 0, len(fields))
	for k := range fields {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func FieldRemove(d *Definition, field string) string {
	for _, p := range []string{"search.fields." + field + ".remove", "fields." + field + ".remove"} {
		if s := str(dotted(d.Raw, p)); s != "" {
			return s
		}
	}
	return ""
}

func LoginFormSelector(d *Definition) string { return str(dotted(d.Raw, "login.form")) }
func LoginSubmitPath(d *Definition) string   { return str(dotted(d.Raw, "login.submitpath")) }
func LoginUsesSelectors(d *Definition) bool {
	return fmt.Sprint(dotted(d.Raw, "login.selectors")) == "true"
}
func LoginTestPath(d *Definition) string     { return str(dotted(d.Raw, "login.test.path")) }
func LoginTestSelector(d *Definition) string { return str(dotted(d.Raw, "login.test.selector")) }
func LoginCaptchaType(d *Definition) string  { return str(dotted(d.Raw, "login.captcha.type")) }

func LoginSelectorInputNames(d *Definition) []string {
	m := mapAny(dotted(d.Raw, "login.selectorinputs"))
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func LoginSelectorInputSelector(d *Definition, name string) string {
	return str(dotted(d.Raw, "login.selectorinputs."+name+".selector"))
}

func LoginSelectorInputAttr(d *Definition, name string) string {
	return str(dotted(d.Raw, "login.selectorinputs."+name+".attribute"))
}

func LoginSelectorInputFilters(d *Definition, name string) []any {
	return slice(dotted(d.Raw, "login.selectorinputs."+name+".filters"))
}

type SelectorSpec struct {
	Selector          string
	Attribute         string
	Filters           []any
	UseBeforeResponse bool
}

func selectorSpec(d *Definition, v any) SelectorSpec {
	m := mapAny(v)
	return SelectorSpec{
		Selector:          Render(anyString(m["selector"]), d.Config, nil, nil),
		Attribute:         anyString(m["attribute"]),
		Filters:           filterArgs(m["filters"]),
		UseBeforeResponse: fmt.Sprint(m["usebeforeresponse"]) == "true",
	}
}

func DownloadSelectors(d *Definition) []SelectorSpec {
	out := []SelectorSpec{}
	for _, v := range slice(dotted(d.Raw, "download.selectors")) {
		sel := selectorSpec(d, v)
		if sel.Selector == "" {
			continue
		}
		out = append(out, sel)
	}
	return out
}

func DetailFieldSelector(d *Definition, field string) string {
	return str(dotted(d.Raw, "details.fields."+field+".selector"))
}

func DetailFieldAttr(d *Definition, field string) string {
	return str(dotted(d.Raw, "details.fields."+field+".attribute"))
}

func DownloadInfoHashSelector(d *Definition, field string) SelectorSpec {
	return selectorSpec(d, dotted(d.Raw, "download.infohash."+field))
}

func DownloadInfoHashUsesBeforeResponse(d *Definition) bool {
	return fmt.Sprint(dotted(d.Raw, "download.infohash.usebeforeresponse")) == "true"
}

func DownloadBeforePathSelector(d *Definition) SelectorSpec {
	return selectorSpec(d, dotted(d.Raw, "download.before.pathselector"))
}

func RowsAttribute(d *Definition) string { return str(dotted(d.Raw, "search.rows.attribute")) }
func RowsRemove(d *Definition) string    { return str(dotted(d.Raw, "search.rows.remove")) }
func RowsMissingAttributeEqualsNoResults(d *Definition) bool {
	return fmt.Sprint(dotted(d.Raw, "search.rows.missingAttributeEqualsNoResults")) == "true"
}
func RowFilters(d *Definition) []any { return slice(dotted(d.Raw, "search.rows.filters")) }
func RowsDateHeaderSelector(d *Definition) string {
	return str(dotted(d.Raw, "search.rows.dateheaders.selector"))
}
func RowsCountSelector(d *Definition) string {
	return str(dotted(d.Raw, "search.rows.count.selector"))
}
func RowsAfter(d *Definition) int {
	n, _ := strconv.Atoi(fmt.Sprint(dotted(d.Raw, "search.rows.after")))
	return n
}

func FieldOptional(d *Definition, field string) bool {
	for _, p := range []string{"search.fields." + field + ".optional", "fields." + field + ".optional"} {
		if fmt.Sprint(dotted(d.Raw, p)) == "true" {
			return true
		}
	}
	return false
}

func FieldDefault(d *Definition, field string, result map[string]string) string {
	if !FieldOptional(d, field) {
		return ""
	}
	for _, p := range []string{"search.fields." + field + ".default", "fields." + field + ".default"} {
		if v := dotted(d.Raw, p); v != nil {
			return Render(anyString(v), d.Config, nil, result)
		}
	}
	return ""
}

func FieldCase(d *Definition, field, val string) string {
	c := mapAny(dotted(d.Raw, "search.fields."+field+".case"))
	if len(c) == 0 {
		c = mapAny(dotted(d.Raw, "fields."+field+".case"))
	}
	if v, ok := c[val]; ok {
		return anyString(v)
	}
	return val
}

type ErrorSelector struct {
	Selector        string
	MessageSelector string
}

func ErrorSelectors(d *Definition) []ErrorSelector {
	out := []ErrorSelector{}
	for _, v := range slice(dotted(d.Raw, "search.error")) {
		m := mapAny(v)
		es := ErrorSelector{Selector: anyString(m["selector"])}
		msg := mapAny(m["message"])
		es.MessageSelector = anyString(msg["selector"])
		if es.Selector != "" {
			out = append(out, es)
		}
	}
	return out
}

func Preprocess(d *Definition, body string) string {
	return ApplyFilterList(d, slice(dotted(d.Raw, "search.preprocessingfilters")), body, nil)
}

func ApplyFilters(d *Definition, field, val string, result map[string]string) string {
	return ApplyFilterList(d, slice(dotted(d.Raw, "search.fields."+field+".filters")), val, result)
}

func ApplyFilterList(d *Definition, filters []any, val string, result map[string]string) string {
	for _, fv := range filters {
		f := mapAny(fv)
		name := str(f["name"])
		args := filterArgs(f["args"])
		base := strings.Split(name, "#")[0]
		switch base {
		case "tolower", "lowercase":
			val = strings.ToLower(val)
		case "toupper", "uppercase":
			val = strings.ToUpper(val)
		case "trim":
			val = strings.TrimSpace(val)
		case "replace":
			if len(args) >= 2 {
				val = strings.ReplaceAll(val, anyString(args[0]), Render(anyString(args[1]), d.Config, nil, result))
			}
		case "re_replace":
			if len(args) >= 2 {
				if re, err := regexp.Compile(anyString(args[0])); err == nil {
					val = re.ReplaceAllString(val, Render(anyString(args[1]), d.Config, nil, result))
				}
			}
		case "regexp":
			if len(args) >= 1 {
				if re, err := regexp.Compile(anyString(args[0])); err == nil {
					if m := re.FindStringSubmatch(val); len(m) > 1 {
						val = m[1]
					} else if len(m) == 1 {
						val = m[0]
					}
				}
			}
		case "split":
			if len(args) >= 2 {
				parts := strings.Split(val, anyString(args[0]))
				if i, err := strconv.Atoi(anyString(args[1])); err == nil && i >= 0 && i < len(parts) {
					val = parts[i]
				}
			}
		case "prepend":
			val = Render(firstArg(args, f["args"]), d.Config, nil, result) + val
		case "append":
			val += Render(firstArg(args, f["args"]), d.Config, nil, result)
		case "urlencode", "querystring":
			val = url.QueryEscape(val)
		case "urldecode":
			if u, err := url.QueryUnescape(val); err == nil {
				val = u
			}
		case "htmldecode":
			val = html.UnescapeString(val)
		case "htmlencode":
			val = html.EscapeString(val)
		case "substring":
			if len(args) >= 2 {
				start, _ := strconv.Atoi(anyString(args[0]))
				ln, _ := strconv.Atoi(anyString(args[1]))
				if start >= 0 && start < len(val) {
					end := start + ln
					if end > len(val) {
						end = len(val)
					}
					val = val[start:end]
				}
			}
		case "striptags", "strip_tags":
			val = regexp.MustCompile(`<[^>]+>`).ReplaceAllString(val, "")
		case "jsonjoinarray":
			sep := ","
			if len(args) > 0 {
				sep = anyString(args[0])
			}
			var arr []any
			if err := json.Unmarshal([]byte(val), &arr); err == nil {
				parts := make([]string, 0, len(arr))
				for _, x := range arr {
					parts = append(parts, anyString(x))
				}
				val = strings.Join(parts, sep)
			}
		case "validate":
			if len(args) >= 1 {
				if re, err := regexp.Compile(anyString(args[0])); err == nil && !re.MatchString(val) {
					val = ""
				}
			}
		case "andmatch":
			if !andMatch(val, args) {
				val = ""
			}
		case "dateparse", "timeparse":
			if len(args) >= 1 {
				if ts, ok := parseByFormat(val, anyString(args[0])); ok {
					val = ts.UTC().Format(time.RFC3339)
				}
			}
		case "timeago", "reltime":
			if ts, ok := parseRelativeTime(val, time.Now().UTC()); ok {
				val = ts.UTC().Format(time.RFC3339)
			}
		case "fuzzytime":
			if ts, ok := parseFuzzyTime(val, time.Now().UTC()); ok {
				val = ts.UTC().Format(time.RFC3339)
			}
		case "diacritics":
			val = stripDiacritics(val)
		case "validfilename":
			val = validFilename(val)
		case "num_add", "add", "num_sub", "sub", "num_mul", "mul", "mult", "num_div", "div":
			if len(args) >= 1 {
				val = mathFilter(val, anyString(args[0]), base)
			}
		}
	}
	return val
}

func filterArgs(v any) []any {
	if v == nil {
		return nil
	}
	if s := slice(v); len(s) > 0 {
		return s
	}
	return []any{v}
}

func firstArg(args []any, raw any) string {
	if len(args) > 0 {
		return anyString(args[0])
	}
	return anyString(raw)
}

func mathFilter(val, arg, op string) string {
	a, err1 := strconv.ParseFloat(strings.TrimSpace(val), 64)
	b, err2 := strconv.ParseFloat(strings.TrimSpace(arg), 64)
	if err1 != nil || err2 != nil {
		return val
	}
	switch op {
	case "num_add", "add":
		a += b
	case "num_sub", "sub":
		a -= b
	case "num_mul", "mul", "mult":
		a *= b
	case "num_div", "div":
		if b != 0 {
			a /= b
		}
	}
	return strconv.FormatFloat(a, 'f', -1, 64)
}

func andMatch(val string, args []any) bool {
	for _, a := range args {
		pat := anyString(a)
		if pat == "" {
			continue
		}
		if re, err := regexp.Compile(pat); err == nil {
			if !re.MatchString(val) {
				return false
			}
		} else if !strings.Contains(val, pat) {
			return false
		}
	}
	return true
}

func parseByFormat(val, fmtSpec string) (time.Time, bool) {
	repl := strings.NewReplacer(
		"yyyy", "2006",
		"MMM", "Jan",
		"MM", "01",
		"dd", "02",
		"HH", "15",
		"hh", "03",
		"mm", "04",
		"ss", "05",
		"zzz", "MST",
		"tt", "PM",
	)
	layout := repl.Replace(fmtSpec)
	for _, candidate := range []string{layout, strings.TrimSpace(layout)} {
		if ts, err := time.Parse(candidate, strings.TrimSpace(val)); err == nil {
			return ts, true
		}
		if ts, err := time.ParseInLocation(candidate, strings.TrimSpace(val), time.UTC); err == nil {
			return ts, true
		}
	}
	return time.Time{}, false
}

func parseRelativeTime(val string, now time.Time) (time.Time, bool) {
	s := strings.ToLower(strings.TrimSpace(val))
	if s == "yesterday" {
		return now.Add(-24 * time.Hour), true
	}
	if s == "today" {
		return now, true
	}
	re := regexp.MustCompile(`^(\d+)\s+(minute|hour|day|week|month|year)s?\s+ago$`)
	if m := re.FindStringSubmatch(s); len(m) == 3 {
		n, _ := strconv.Atoi(m[1])
		switch m[2] {
		case "minute":
			return now.Add(-time.Duration(n) * time.Minute), true
		case "hour":
			return now.Add(-time.Duration(n) * time.Hour), true
		case "day":
			return now.AddDate(0, 0, -n), true
		case "week":
			return now.AddDate(0, 0, -7*n), true
		case "month":
			return now.AddDate(0, -n, 0), true
		case "year":
			return now.AddDate(-n, 0, 0), true
		}
	}
	return time.Time{}, false
}

func parseFuzzyTime(val string, now time.Time) (time.Time, bool) {
	s := strings.TrimSpace(val)
	if ts, ok := parseRelativeTime(s, now); ok {
		return ts, true
	}
	for _, layout := range []string{"2006-01-02", time.RFC1123, time.RFC1123Z, "02 Jan 2006 15:04", "Jan 02 2006 15:04"} {
		if ts, err := time.Parse(layout, s); err == nil {
			return ts, true
		}
	}
	for _, prefix := range []struct {
		P string
		D int
	}{{"Yesterday ", -1}, {"Today ", 0}} {
		if strings.HasPrefix(s, prefix.P) {
			if tm, err := time.Parse("15:04", strings.TrimPrefix(s, prefix.P)); err == nil {
				base := now.AddDate(0, 0, prefix.D)
				return time.Date(base.Year(), base.Month(), base.Day(), tm.Hour(), tm.Minute(), 0, 0, time.UTC), true
			}
		}
	}
	return time.Time{}, false
}

var diacriticsReplacer = strings.NewReplacer(
	"à", "a", "á", "a", "â", "a", "ã", "a", "ä", "a", "å", "a",
	"ç", "c",
	"è", "e", "é", "e", "ê", "e", "ë", "e",
	"ì", "i", "í", "i", "î", "i", "ï", "i",
	"ñ", "n",
	"ò", "o", "ó", "o", "ô", "o", "õ", "o", "ö", "o",
	"ù", "u", "ú", "u", "û", "u", "ü", "u",
	"ý", "y", "ÿ", "y",
	"À", "A", "Á", "A", "Â", "A", "Ã", "A", "Ä", "A", "Å", "A",
	"Ç", "C",
	"È", "E", "É", "E", "Ê", "E", "Ë", "E",
	"Ì", "I", "Í", "I", "Î", "I", "Ï", "I",
	"Ñ", "N",
	"Ò", "O", "Ó", "O", "Ô", "O", "Õ", "O", "Ö", "O",
	"Ù", "U", "Ú", "U", "Û", "U", "Ü", "U",
	"Ý", "Y",
	"œ", "oe", "Œ", "OE", "æ", "ae", "Æ", "AE",
	"ß", "ss",
	"š", "s", "Š", "S", "ž", "z", "Ž", "Z",
	"ř", "r", "Ř", "R", "č", "c", "Č", "C",
	"ď", "d", "Ď", "D", "ť", "t", "Ť", "T",
	"ľ", "l", "Ľ", "L", "ĺ", "l", "Ĺ", "L",
	"ń", "n", "Ń", "N",
	"ğ", "g", "Ğ", "G",
	"ş", "s", "Ş", "S",
	"ı", "i",
	"ł", "l", "Ł", "L",
	"ą", "a", "Ą", "A", "ę", "e", "Ę", "E",
	"ś", "s", "Ś", "S", "ź", "z", "Ź", "Z", "ż", "z", "Ż", "Z",
	"ø", "o", "Ø", "O",
	"ð", "d", "Ð", "D",
	"þ", "th", "Þ", "Th",
	"ğ", "g",
)

func stripDiacritics(s string) string { return diacriticsReplacer.Replace(s) }

func validFilename(s string) string {
	invalid := regexp.MustCompile(`[<>:"/\\|?*\x00-\x1f]`)
	s = invalid.ReplaceAllString(s, "_")
	s = strings.TrimSpace(s)
	for strings.Contains(s, "__") {
		s = strings.ReplaceAll(s, "__", "_")
	}
	return s
}

func renderedMethod(t string, cfg, query, result map[string]string) string {
	return strings.ToLower(strings.TrimSpace(Render(t, cfg, query, result)))
}

func Render(t string, cfg, query, result map[string]string) string {
	if cfg == nil {
		cfg = map[string]string{}
	}
	if query == nil {
		query = map[string]string{}
	}
	if result == nil {
		result = map[string]string{}
	}
	reIf := regexp.MustCompile(`(?s){{\s*if\s+([^}]+?)\s*}}(.*?)({{\s*else\s*}}(.*?))?{{\s*end\s*}}`)
	for {
		old := t
		t = reIf.ReplaceAllStringFunc(t, func(block string) string {
			m := reIf.FindStringSubmatch(block)
			if evalBool(strings.TrimSpace(m[1]), cfg, query, result) {
				return m[2]
			}
			if len(m) > 4 {
				return m[4]
			}
			return ""
		})
		if t == old {
			break
		}
	}
	reJoin := regexp.MustCompile(`{{\s*join\s+\.Categories\s+"([^"]*)"\s*}}`)
	t = reJoin.ReplaceAllStringFunc(t, func(tag string) string {
		m := reJoin.FindStringSubmatch(tag)
		return strings.Join(splitCSV(query["Categories"]), m[1])
	})
	reVar := regexp.MustCompile(`{{\s*([^}]+?)\s*}}`)
	return reVar.ReplaceAllStringFunc(t, func(tag string) string {
		m := reVar.FindStringSubmatch(tag)
		return renderExpr(strings.TrimSpace(m[1]), cfg, query, result)
	})
}

func renderExpr(expr string, cfg, query, result map[string]string) string {
	if m := regexp.MustCompile(`^re_replace\s+([^\s]+)\s+"([^"]*)"\s+"([^"]*)"$`).FindStringSubmatch(expr); len(m) == 4 {
		val := value(m[1], cfg, query, result)
		if re, err := regexp.Compile(m[2]); err == nil {
			return re.ReplaceAllString(val, m[3])
		}
		return val
	}
	if m := regexp.MustCompile(`^replace\s+([^\s]+)\s+"([^"]*)"\s+"([^"]*)"$`).FindStringSubmatch(expr); len(m) == 4 {
		return strings.ReplaceAll(value(m[1], cfg, query, result), m[2], m[3])
	}
	return value(expr, cfg, query, result)
}

func value(expr string, cfg, query, result map[string]string) string {
	expr = strings.Trim(expr, " ")
	switch {
	case expr == ".Keywords":
		return query["Keywords"]
	case expr == ".Query", expr == ".Query.Q":
		return query["Query"]
	case expr == ".Query.Keywords":
		return query["Keywords"]
	case strings.HasPrefix(expr, ".Query."):
		return query[strings.TrimPrefix(expr, ".Query.")]
	case expr == ".True":
		return "true"
	case expr == ".False":
		return "false"
	case strings.HasPrefix(expr, ".Config."):
		return cfg[strings.TrimPrefix(expr, ".Config.")]
	case strings.HasPrefix(expr, ".Result."):
		return result[strings.TrimPrefix(expr, ".Result.")]
	case strings.HasPrefix(expr, `"`) && strings.HasSuffix(expr, `"`):
		return strings.Trim(expr, `"`)
	case strings.HasPrefix(expr, "."):
		return query[strings.TrimPrefix(expr, ".")]
	}
	return ""
}

func splitCSV(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if strings.TrimSpace(p) != "" {
			out = append(out, strings.TrimSpace(p))
		}
	}
	return out
}

func truthy(s string) bool { return s != "" && s != "0" && strings.ToLower(s) != "false" }

func evalBool(expr string, cfg, query, result map[string]string) bool {
	expr = strings.TrimSpace(expr)
	if strings.HasPrefix(expr, "(") && strings.HasSuffix(expr, ")") {
		return evalBool(expr[1:len(expr)-1], cfg, query, result)
	}
	parts := splitExpr(expr)
	if len(parts) == 0 {
		return false
	}
	switch parts[0] {
	case "and":
		for _, p := range parts[1:] {
			if !evalBool(p, cfg, query, result) {
				return false
			}
		}
		return len(parts) > 1
	case "or":
		for _, p := range parts[1:] {
			if evalBool(p, cfg, query, result) {
				return true
			}
		}
		return false
	case "eq", "ne":
		if len(parts) < 3 {
			return false
		}
		a, b := value(parts[1], cfg, query, result), value(parts[2], cfg, query, result)
		if parts[0] == "eq" {
			return a == b
		}
		return a != b
	default:
		return truthy(value(expr, cfg, query, result))
	}
}

func splitExpr(s string) []string {
	var out []string
	var b strings.Builder
	inQuote, depth := false, 0
	for _, r := range s {
		switch r {
		case '"':
			inQuote = !inQuote
			b.WriteRune(r)
		case '(':
			depth++
			b.WriteRune(r)
		case ')':
			depth--
			b.WriteRune(r)
		case ' ', '\t':
			if inQuote || depth > 0 {
				b.WriteRune(r)
			} else if b.Len() > 0 {
				out = append(out, b.String())
				b.Reset()
			}
		default:
			b.WriteRune(r)
		}
	}
	if b.Len() > 0 {
		out = append(out, b.String())
	}
	return out
}

func settingDefaults(raw map[string]any) map[string]string {
	out := map[string]string{}
	for _, v := range slice(raw["settings"]) {
		m := mapAny(v)
		name := str(m["name"])
		if name != "" {
			out[name] = anyString(m["default"])
		}
	}
	return out
}

func str(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
func anyString(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprint(v)
}
func firstAny(vals ...any) any {
	for _, v := range vals {
		if anyString(v) != "" {
			return v
		}
	}
	return nil
}
func first(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
func firstStr(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if s := str(m[k]); s != "" {
			return s
		}
	}
	return ""
}
func mapAny(v any) map[string]any {
	if m, ok := v.(map[string]any); ok {
		return m
	}
	return map[string]any{}
}
func slice(v any) []any {
	if s, ok := v.([]any); ok {
		return s
	}
	return nil
}
func dotted(m map[string]any, p string) any {
	var cur any = m
	for _, part := range strings.Split(p, ".") {
		mm := mapAny(cur)
		cur = mm[part]
		if cur == nil {
			return nil
		}
	}
	return cur
}

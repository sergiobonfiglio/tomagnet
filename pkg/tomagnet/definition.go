package tomagnet

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/sergiobonfiglio/tomagnet/internal/cardigann"
	"gopkg.in/yaml.v3"
)

type Definition struct {
	ID             string             `yaml:"id"`
	Name           string             `yaml:"name"`
	BaseURL        string             `yaml:"base_url"`
	Site           string             `yaml:"site"`
	URL            string             `yaml:"url"`
	Links          []string           `yaml:"links"`
	LegacyLinks    []string           `yaml:"legacylinks"`
	FollowRedirect bool               `yaml:"followredirect"`
	Certificates   []string           `yaml:"certificates"`
	RequestDelay   string             `yaml:"requestDelay"`
	Encoding       string             `yaml:"encoding"`
	Settings       []Setting          `yaml:"settings"`
	Caps           Caps               `yaml:"caps"`
	Search         SearchDefinition   `yaml:"search"`
	Login          LoginDefinition    `yaml:"login"`
	Download       DownloadDefinition `yaml:"download"`
	Details        DetailsDefinition  `yaml:"details"`
}

type Setting struct {
	Name    string `yaml:"name"`
	Default string `yaml:"default"`
}

type Caps struct {
	CategoryMappings []CategoryMapping     `yaml:"categorymappings"`
	Modes            map[string]SearchMode `yaml:"modes"`
}

type CategoryMapping struct {
	ID      string `yaml:"id"`
	Cat     string `yaml:"cat"`
	Default bool   `yaml:"default"`
}

type SearchMode struct {
	Params []SearchParam `yaml:"params"`
}

// UnmarshalYAML accepts both the typed shape tomagnet exposes publicly
// (`params: [{name: ...}]`) and the shorthand array form used by many Jackett
// definitions (`search: [q, imdbid]`). This keeps Jackett compatibility inside
// tomagnet instead of requiring callers to normalize YAML before decoding.
func (s *SearchMode) UnmarshalYAML(value *yaml.Node) error {
	type rawSearchMode SearchMode
	if value.Kind == yaml.SequenceNode {
		var params []string
		if err := value.Decode(&params); err != nil {
			return err
		}
		s.Params = make([]SearchParam, 0, len(params))
		for _, param := range params {
			s.Params = append(s.Params, SearchParam{Name: param})
		}
		return nil
	}
	var aux rawSearchMode
	if err := value.Decode(&aux); err != nil {
		return err
	}
	*s = SearchMode(aux)
	return nil
}

type SearchParam struct {
	Name string `yaml:"name"`
}

type StringMap map[string]string

// UnmarshalYAML accepts either scalar values or single-/multi-value YAML lists
// for map entries. Jackett commonly encodes headers as `Header: ["value"]`.
// Tomagnet keeps a string-valued public API and uses the first list element.
func (m *StringMap) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind != yaml.MappingNode {
		return fmt.Errorf("expected mapping, got %v", value.Kind)
	}
	out := make(StringMap, len(value.Content)/2)
	for i := 0; i < len(value.Content); i += 2 {
		key := value.Content[i].Value
		val := value.Content[i+1]
		switch val.Kind {
		case yaml.SequenceNode:
			if len(val.Content) == 0 {
				out[key] = ""
				continue
			}
			out[key] = val.Content[0].Value
		default:
			out[key] = val.Value
		}
	}
	*m = out
	return nil
}

type CookieMap map[string]string

// UnmarshalYAML accepts both mapping cookies (`uid: ...`) and the Jackett
// shorthand sequence form (`cookies: ["JAVA=OK"]`). Sequence items are parsed
// as `name=value` cookie pairs.
func (m *CookieMap) UnmarshalYAML(value *yaml.Node) error {
	switch value.Kind {
	case yaml.MappingNode:
		var out StringMap
		if err := value.Decode(&out); err != nil {
			return err
		}
		*m = CookieMap(out)
		return nil
	case yaml.SequenceNode:
		out := make(CookieMap, len(value.Content))
		for _, item := range value.Content {
			cookie := item.Value
			name, val, ok := strings.Cut(cookie, "=")
			if !ok {
				out[cookie] = ""
				continue
			}
			out[name] = val
		}
		*m = out
		return nil
	default:
		return fmt.Errorf("expected mapping or sequence, got %v", value.Kind)
	}
}

type SearchDefinition struct {
	Path                 string                     `yaml:"path"`
	Paths                []SearchPath               `yaml:"paths"`
	Method               string                     `yaml:"method"`
	Inputs               StringMap                  `yaml:"inputs"`
	Headers              StringMap                  `yaml:"headers"`
	Cookies              CookieMap                  `yaml:"cookies"`
	Rows                 RowsDefinition             `yaml:"rows"`
	Fields               map[string]FieldDefinition `yaml:"fields"`
	Error                []ErrorSelector            `yaml:"error"`
	PreprocessingFilters []Filter                   `yaml:"preprocessingfilters"`
	KeywordsFilters      []Filter                   `yaml:"keywordsfilters"`
	AllowEmptyInputs     bool                       `yaml:"allowEmptyInputs"`
}

type SearchPath struct {
	Path           string       `yaml:"path"`
	Method         string       `yaml:"method"`
	Inputs         StringMap    `yaml:"inputs"`
	Categories     []string     `yaml:"categories"`
	Response       ResponseSpec `yaml:"response"`
	FollowRedirect bool         `yaml:"followredirect"`
	InheritInputs  *bool        `yaml:"inheritinputs"`
}

type ResponseSpec struct {
	Type string `yaml:"type"`
}

type RowsDefinition struct {
	Selector                        string   `yaml:"selector"`
	Attribute                       string   `yaml:"attribute"`
	Remove                          string   `yaml:"remove"`
	MissingAttributeEqualsNoResults bool     `yaml:"missingAttributeEqualsNoResults"`
	Filters                         []Filter `yaml:"filters"`
	DateHeaders                     Selector `yaml:"dateheaders"`
	Count                           Selector `yaml:"count"`
	After                           int      `yaml:"after"`
}

type FieldDefinition struct {
	Selector  string            `yaml:"selector"`
	Attribute string            `yaml:"attribute"`
	Text      string            `yaml:"text"`
	Remove    string            `yaml:"remove"`
	Optional  bool              `yaml:"optional"`
	Default   string            `yaml:"default"`
	Case      map[string]string `yaml:"case"`
	Filters   []Filter          `yaml:"filters"`
}

type LoginDefinition struct {
	Method         string                   `yaml:"method"`
	Path           string                   `yaml:"path"`
	Form           string                   `yaml:"form"`
	SubmitPath     string                   `yaml:"submitpath"`
	Selectors      bool                     `yaml:"selectors"`
	Inputs         StringMap                `yaml:"inputs"`
	Headers        StringMap                `yaml:"headers"`
	Cookies        CookieMap                `yaml:"cookies"`
	Test           LoginTest                `yaml:"test"`
	Captcha        LoginCaptcha             `yaml:"captcha"`
	SelectorInputs map[string]SelectorInput `yaml:"selectorinputs"`
}

type LoginTest struct {
	Path     string `yaml:"path"`
	Selector string `yaml:"selector"`
}

type LoginCaptcha struct {
	Type string `yaml:"type"`
}

type SelectorInput struct {
	Selector  string   `yaml:"selector"`
	Attribute string   `yaml:"attribute"`
	Filters   []Filter `yaml:"filters"`
}

type DownloadDefinition struct {
	Selectors []SelectorSpec        `yaml:"selectors"`
	InfoHash  DownloadInfoHash      `yaml:"infohash"`
	Before    DownloadBeforeRequest `yaml:"before"`
}

type DownloadInfoHash struct {
	Hash              SelectorSpec `yaml:"hash"`
	UseBeforeResponse bool         `yaml:"usebeforeresponse"`
}

type DownloadBeforeRequest struct {
	PathSelector SelectorSpec `yaml:"pathselector"`
	Method       string       `yaml:"method"`
	Path         string       `yaml:"path"`
	Inputs       StringMap    `yaml:"inputs"`
	Headers      StringMap    `yaml:"headers"`
}

type DetailsDefinition struct {
	Fields map[string]DetailField `yaml:"fields"`
}

type DetailField struct {
	Selector  string `yaml:"selector"`
	Attribute string `yaml:"attribute"`
}

type Selector struct {
	Selector string `yaml:"selector"`
}

type SelectorSpec struct {
	Selector          string   `yaml:"selector"`
	Attribute         string   `yaml:"attribute"`
	Filters           []Filter `yaml:"filters"`
	UseBeforeResponse bool     `yaml:"usebeforeresponse"`
}

type ErrorSelector struct {
	Selector string       `yaml:"selector"`
	Message  ErrorMessage `yaml:"message"`
}

type ErrorMessage struct {
	Selector string `yaml:"selector"`
}

type Filter struct {
	Name string   `yaml:"name"`
	Args []string `yaml:"args"`
}

// UnmarshalYAML accepts both the typed args form (`args: [":", ""]`) and
// the scalar shorthand used by some Jackett definitions (`args: " suffix"`).
// Scalar args are normalized to a single-element slice so the rest of tomagnet
// can keep using a stable typed API.
func (f *Filter) UnmarshalYAML(value *yaml.Node) error {
	var aux struct {
		Name string `yaml:"name"`
		Args any    `yaml:"args"`
	}
	if err := value.Decode(&aux); err != nil {
		return err
	}
	f.Name = aux.Name
	switch args := aux.Args.(type) {
	case nil:
		f.Args = nil
	case []any:
		f.Args = make([]string, 0, len(args))
		for _, arg := range args {
			f.Args = append(f.Args, fmt.Sprint(arg))
		}
	default:
		f.Args = []string{fmt.Sprint(args)}
	}
	return nil
}

func DecodeDefinition(r io.Reader) (*Definition, error) {
	b, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return decodeDefinitionBytes(b)
}

func LoadDefinition(path string) (*Definition, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return decodeDefinitionBytes(b)
}

func decodeDefinitionBytes(b []byte) (*Definition, error) {
	var d Definition
	if err := yaml.NewDecoder(bytes.NewReader(b)).Decode(&d); err == nil {
		return &d, nil
	}
	fixed := []byte(strings.ReplaceAll(string(b), `\/`, `/`))
	if err := yaml.NewDecoder(bytes.NewReader(fixed)).Decode(&d); err != nil {
		return nil, err
	}
	return &d, nil
}

func (d Definition) cardigann() *cardigann.Definition {
	return cardigann.FromRaw(d.raw())
}

func (d Definition) raw() map[string]any {
	m := map[string]any{}
	put(m, "id", d.ID)
	put(m, "name", d.Name)
	put(m, "base_url", d.BaseURL)
	put(m, "site", d.Site)
	put(m, "url", d.URL)
	put(m, "links", stringSlice(d.Links))
	put(m, "legacylinks", stringSlice(d.LegacyLinks))
	putBool(m, "followredirect", d.FollowRedirect)
	put(m, "certificates", stringSlice(d.Certificates))
	put(m, "requestDelay", d.RequestDelay)
	put(m, "encoding", d.Encoding)
	if len(d.Settings) > 0 {
		m["settings"] = settingsRaw(d.Settings)
	}
	if caps := d.Caps.raw(); len(caps) > 0 {
		m["caps"] = caps
	}
	if search := d.Search.raw(); len(search) > 0 {
		m["search"] = search
	}
	if login := d.Login.raw(); len(login) > 0 {
		m["login"] = login
	}
	if download := d.Download.raw(); len(download) > 0 {
		m["download"] = download
	}
	if details := d.Details.raw(); len(details) > 0 {
		m["details"] = details
	}
	return m
}

func settingsRaw(in []Setting) []any {
	out := make([]any, 0, len(in))
	for _, v := range in {
		m := map[string]any{}
		put(m, "name", v.Name)
		put(m, "default", v.Default)
		out = append(out, m)
	}
	return out
}
func (c Caps) raw() map[string]any {
	m := map[string]any{}
	if len(c.CategoryMappings) > 0 {
		a := []any{}
		for _, v := range c.CategoryMappings {
			mm := map[string]any{}
			put(mm, "id", v.ID)
			put(mm, "cat", v.Cat)
			putBool(mm, "default", v.Default)
			a = append(a, mm)
		}
		m["categorymappings"] = a
	}
	if len(c.Modes) > 0 {
		modes := map[string]any{}
		for k, v := range c.Modes {
			modes[k] = v.raw()
		}
		m["modes"] = modes
	}
	return m
}
func (s SearchMode) raw() map[string]any {
	m := map[string]any{}
	if len(s.Params) > 0 {
		a := []any{}
		for _, p := range s.Params {
			mm := map[string]any{}
			put(mm, "name", p.Name)
			a = append(a, mm)
		}
		m["params"] = a
	}
	return m
}
func (s SearchDefinition) raw() map[string]any {
	m := map[string]any{}
	put(m, "path", s.Path)
	if len(s.Paths) > 0 {
		a := []any{}
		for _, p := range s.Paths {
			a = append(a, p.raw())
		}
		m["paths"] = a
	}
	put(m, "method", s.Method)
	putMap(m, "inputs", s.Inputs)
	putMap(m, "headers", s.Headers)
	putMap(m, "cookies", s.Cookies)
	if rows := s.Rows.raw(); len(rows) > 0 {
		m["rows"] = rows
	}
	putFields(m, "fields", s.Fields)
	if len(s.Error) > 0 {
		a := []any{}
		for _, e := range s.Error {
			a = append(a, e.raw())
		}
		m["error"] = a
	}
	putFilters(m, "preprocessingfilters", s.PreprocessingFilters)
	putFilters(m, "keywordsfilters", s.KeywordsFilters)
	putBool(m, "allowEmptyInputs", s.AllowEmptyInputs)
	return m
}
func (p SearchPath) raw() map[string]any {
	m := map[string]any{}
	put(m, "path", p.Path)
	put(m, "method", p.Method)
	putMap(m, "inputs", p.Inputs)
	put(m, "categories", stringSlice(p.Categories))
	if r := p.Response.raw(); len(r) > 0 {
		m["response"] = r
	}
	putBool(m, "followredirect", p.FollowRedirect)
	if p.InheritInputs != nil {
		m["inheritinputs"] = *p.InheritInputs
	}
	return m
}
func (r ResponseSpec) raw() map[string]any { m := map[string]any{}; put(m, "type", r.Type); return m }
func (r RowsDefinition) raw() map[string]any {
	m := map[string]any{}
	put(m, "selector", r.Selector)
	put(m, "attribute", r.Attribute)
	put(m, "remove", r.Remove)
	putBool(m, "missingAttributeEqualsNoResults", r.MissingAttributeEqualsNoResults)
	putFilters(m, "filters", r.Filters)
	if r.DateHeaders.Selector != "" {
		m["dateheaders"] = map[string]any{"selector": r.DateHeaders.Selector}
	}
	if r.Count.Selector != "" {
		m["count"] = map[string]any{"selector": r.Count.Selector}
	}
	if r.After != 0 {
		m["after"] = r.After
	}
	return m
}
func (l LoginDefinition) raw() map[string]any {
	m := map[string]any{}
	put(m, "method", l.Method)
	put(m, "path", l.Path)
	put(m, "form", l.Form)
	put(m, "submitpath", l.SubmitPath)
	putBool(m, "selectors", l.Selectors)
	putMap(m, "inputs", l.Inputs)
	putMap(m, "headers", l.Headers)
	putMap(m, "cookies", l.Cookies)
	if t := l.Test.raw(); len(t) > 0 {
		m["test"] = t
	}
	if l.Captcha.Type != "" {
		m["captcha"] = map[string]any{"type": l.Captcha.Type}
	}
	if len(l.SelectorInputs) > 0 {
		mm := map[string]any{}
		for k, v := range l.SelectorInputs {
			mm[k] = v.raw()
		}
		m["selectorinputs"] = mm
	}
	return m
}
func (t LoginTest) raw() map[string]any {
	m := map[string]any{}
	put(m, "path", t.Path)
	put(m, "selector", t.Selector)
	return m
}
func (s SelectorInput) raw() map[string]any {
	m := map[string]any{}
	put(m, "selector", s.Selector)
	put(m, "attribute", s.Attribute)
	putFilters(m, "filters", s.Filters)
	return m
}
func (d DownloadDefinition) raw() map[string]any {
	m := map[string]any{}
	if len(d.Selectors) > 0 {
		a := []any{}
		for _, s := range d.Selectors {
			a = append(a, s.raw())
		}
		m["selectors"] = a
	}
	if ih := d.InfoHash.raw(); len(ih) > 0 {
		m["infohash"] = ih
	}
	if b := d.Before.raw(); len(b) > 0 {
		m["before"] = b
	}
	return m
}
func (i DownloadInfoHash) raw() map[string]any {
	m := map[string]any{}
	if h := i.Hash.raw(); len(h) > 0 {
		m["hash"] = h
	}
	putBool(m, "usebeforeresponse", i.UseBeforeResponse)
	return m
}
func (b DownloadBeforeRequest) raw() map[string]any {
	m := map[string]any{}
	if ps := b.PathSelector.raw(); len(ps) > 0 {
		m["pathselector"] = ps
	}
	put(m, "method", b.Method)
	put(m, "path", b.Path)
	putMap(m, "inputs", b.Inputs)
	putMap(m, "headers", b.Headers)
	return m
}
func (d DetailsDefinition) raw() map[string]any {
	m := map[string]any{}
	if len(d.Fields) > 0 {
		fields := map[string]any{}
		for k, v := range d.Fields {
			mm := map[string]any{}
			put(mm, "selector", v.Selector)
			put(mm, "attribute", v.Attribute)
			fields[k] = mm
		}
		m["fields"] = fields
	}
	return m
}
func (s SelectorSpec) raw() map[string]any {
	m := map[string]any{}
	put(m, "selector", s.Selector)
	put(m, "attribute", s.Attribute)
	putFilters(m, "filters", s.Filters)
	putBool(m, "usebeforeresponse", s.UseBeforeResponse)
	return m
}
func (e ErrorSelector) raw() map[string]any {
	m := map[string]any{}
	put(m, "selector", e.Selector)
	if e.Message.Selector != "" {
		m["message"] = map[string]any{"selector": e.Message.Selector}
	}
	return m
}
func (f Filter) raw() map[string]any {
	m := map[string]any{}
	put(m, "name", f.Name)
	put(m, "args", stringSlice(f.Args))
	return m
}
func putFields(m map[string]any, k string, fields map[string]FieldDefinition) {
	if len(fields) == 0 {
		return
	}
	out := map[string]any{}
	for name, f := range fields {
		mm := map[string]any{}
		put(mm, "selector", f.Selector)
		put(mm, "attribute", f.Attribute)
		put(mm, "text", f.Text)
		put(mm, "remove", f.Remove)
		putBool(mm, "optional", f.Optional)
		put(mm, "default", f.Default)
		putMap(mm, "case", f.Case)
		putFilters(mm, "filters", f.Filters)
		out[name] = mm
	}
	m[k] = out
}
func putFilters(m map[string]any, k string, filters []Filter) {
	if len(filters) == 0 {
		return
	}
	out := []any{}
	for _, f := range filters {
		out = append(out, f.raw())
	}
	m[k] = out
}
func putMap(m map[string]any, k string, v any) {
	var src map[string]string
	switch x := v.(type) {
	case map[string]string:
		src = x
	case StringMap:
		src = map[string]string(x)
	case CookieMap:
		src = map[string]string(x)
	default:
		return
	}
	if len(src) == 0 {
		return
	}
	out := map[string]any{}
	for kk, vv := range src {
		out[kk] = vv
	}
	m[k] = out
}
func put(m map[string]any, k string, v any) {
	switch x := v.(type) {
	case string:
		if x != "" {
			m[k] = x
		}
	case []any:
		if len(x) > 0 {
			m[k] = x
		}
	}
}
func putBool(m map[string]any, k string, v bool) {
	if v {
		m[k] = v
	}
}
func stringSlice(in []string) []any {
	if len(in) == 0 {
		return nil
	}
	out := make([]any, len(in))
	for i, v := range in {
		out[i] = v
	}
	return out
}

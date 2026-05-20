package cardigann

import (
	"fmt"
	"sort"
	"strings"
)

func CheckSupport(d *Definition) error {
	bad := map[string]bool{}
	add := func(s string) {
		if s != "" {
			bad[s] = true
		}
	}
	for k := range d.Raw {
		if !topSupport[k] {
			add("top." + k)
		}
	}
	search := mapAny(d.Raw["search"])
	for k := range search {
		if !searchSupport[k] {
			add("search." + k)
		}
	}
	for _, p := range slice(search["paths"]) {
		for k := range mapAny(p) {
			if !searchPathSupport[k] {
				add("search.paths." + k)
			}
		}
	}
	for k := range mapAny(search["rows"]) {
		if !rowSupport[k] {
			add("search.rows." + k)
		}
	}
	for field, fv := range mapAny(search["fields"]) {
		for k := range mapAny(fv) {
			if !fieldSupport[k] {
				add("search.fields." + field + "." + k)
			}
		}
		for _, ff := range slice(mapAny(fv)["filters"]) {
			name := strings.Split(anyString(mapAny(ff)["name"]), "#")[0]
			if !filterSupport[name] {
				add("filter." + name)
			}
		}
	}
	for _, ff := range slice(search["keywordsfilters"]) {
		name := strings.Split(anyString(mapAny(ff)["name"]), "#")[0]
		if !filterSupport[name] {
			add("filter." + name)
		}
	}
	for _, ff := range slice(mapAny(search["rows"])["filters"]) {
		name := strings.Split(anyString(mapAny(ff)["name"]), "#")[0]
		if !filterSupport[name] {
			add("filter." + name)
		}
	}
	login := mapAny(d.Raw["login"])
	for k := range login {
		if !loginSupport[k] {
			add("login." + k)
		}
	}
	download := mapAny(d.Raw["download"])
	for k := range download {
		if !downloadSupport[k] {
			add("download." + k)
		}
	}
	if len(bad) == 0 {
		return nil
	}
	list := make([]string, 0, len(bad))
	for s := range bad {
		list = append(list, s)
	}
	sort.Strings(list)
	return fmt.Errorf("%s", strings.Join(list, ", "))
}

var topSupport = map[string]bool{
	"caps": true, "description": true, "download": true, "encoding": true, "followredirect": true,
	"id": true, "language": true, "legacylinks": true, "links": true, "login": true, "name": true,
	"replaces": true, "requestDelay": true, "search": true, "settings": true, "testlinktorrent": true,
	"type": true, "certificates": true,
}
var searchSupport = map[string]bool{
	"allowEmptyInputs": true, "error": true, "fields": true, "headers": true, "inputs": true,
	"keywordsfilters": true, "path": true, "paths": true, "rows": true,
}
var searchPathSupport = map[string]bool{
	"categories": true, "followredirect": true, "inheritinputs": true, "inputs": true, "method": true,
	"path": true, "response": true,
}
var rowSupport = map[string]bool{
	"after": true, "attribute": true, "count": true, "dateheaders": true, "filters": true,
	"missingAttributeEqualsNoResults": true, "multiple": true, "remove": true, "selector": true,
}
var fieldSupport = map[string]bool{
	"attribute": true, "case": true, "default": true, "filters": true, "optional": true,
	"remove": true, "selector": true, "text": true,
}
var loginSupport = map[string]bool{
	"captcha": true, "cookies": true, "error": true, "form": true, "headers": true, "inputs": true,
	"method": true, "path": true, "selectorinputs": true, "selectors": true, "submitpath": true, "test": true,
}
var downloadSupport = map[string]bool{
	"before": true, "infohash": true, "method": true, "selectors": true,
}
var filterSupport = map[string]bool{
	"add": true, "andmatch": true, "append": true, "dateparse": true, "diacritics": true, "div": true,
	"fuzzytime": true, "htmldecode": true, "htmlencode": true, "jsonjoinarray": true, "lowercase": true,
	"mul": true, "mult": true, "num_add": true, "num_div": true, "num_mul": true, "num_sub": true,
	"prepend": true, "querystring": true, "re_replace": true, "regexp": true, "replace": true,
	"split": true, "strip_tags": true, "striptags": true, "sub": true, "timeago": true,
	"timeparse": true, "tolower": true, "toupper": true, "trim": true, "uppercase": true,
	"urldecode": true, "urlencode": true, "validate": true, "validfilename": true,
}

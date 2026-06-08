package search

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/sergiobonfiglio/tomagnet/internal/cardigann"
	"github.com/sergiobonfiglio/tomagnet/internal/fetch"
)

func buildLoginRequest(d *cardigann.Definition, page string) cardigann.RequestSpec {
	spec := cardigann.LoginRequest(d)
	formSel := cardigann.LoginFormSelector(d)
	submitPath := cardigann.LoginSubmitPath(d)
	selectorNames := cardigann.LoginSelectorInputNames(d)
	if formSel == "" && submitPath == "" && len(selectorNames) == 0 && spec.Method != "form" {
		return spec
	}
	if spec.Method == "form" {
		spec.Method = "post"
	}
	if strings.TrimSpace(page) == "" {
		if spec.Method == "form" {
			spec.Method = "post"
		}
		if submitPath != "" {
			spec.Path = submitPath
		}
		return spec
	}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(page))
	if err != nil {
		if submitPath != "" {
			spec.Path = submitPath
		}
		return spec
	}
	scope := doc.Selection
	if formSel != "" {
		form := selectNodes(doc.Selection, formSel).First()
		if form.Length() > 0 {
			scope = form
			if submitPath == "" {
				if action, ok := form.Attr("action"); ok && strings.TrimSpace(action) != "" {
					spec.Path = action
				}
			}
		}
	}
	if submitPath != "" {
		spec.Path = submitPath
	}
	if cardigann.LoginUsesSelectors(d) {
		resolved := map[string]string{}
		for k, v := range spec.Inputs {
			q := selectNodes(scope, k).First()
			if q.Length() == 0 {
				q = selectNodes(doc.Selection, k).First()
			}
			if q.Length() == 0 {
				resolved[k] = v
				continue
			}
			name, ok := q.Attr("name")
			if !ok || strings.TrimSpace(name) == "" {
				resolved[k] = v
				continue
			}
			resolved[name] = v
		}
		spec.Inputs = resolved
	}
	for _, name := range selectorNames {
		sel := cardigann.LoginSelectorInputSelector(d, name)
		q := selectNodes(scope, sel).First()
		if q.Length() == 0 {
			q = selectNodes(doc.Selection, sel).First()
		}
		if q.Length() == 0 {
			continue
		}
		val := ""
		if attr := cardigann.LoginSelectorInputAttr(d, name); attr != "" {
			val, _ = q.Attr(attr)
		} else {
			val = q.Text()
		}
		val = cardigann.ApplyFilterList(d, cardigann.LoginSelectorInputFilters(d, name), val, nil)
		if val != "" {
			spec.Inputs[name] = val
		}
	}
	return spec
}

func loginNeedsPage(d *cardigann.Definition, spec cardigann.RequestSpec) bool {
	return spec.Method == "form" || cardigann.LoginFormSelector(d) != "" || cardigann.LoginSubmitPath(d) != "" || len(cardigann.LoginSelectorInputNames(d)) > 0 || cardigann.LoginUsesSelectors(d)
}

func mergeCookies(dst, src map[string]string) map[string]string {
	if dst == nil {
		dst = map[string]string{}
	}
	maps.Copy(dst, src)
	return dst
}

func loginCookieHeader(spec cardigann.RequestSpec) string {
	if spec.Headers["Cookie"] != "" {
		return spec.Headers["Cookie"]
	}
	return strings.TrimSpace(spec.Inputs["cookie"])
}

var errLoginTestFailed = errors.New("login test failed")

func loginUnsupported(d *cardigann.Definition) error {
	if d == nil {
		return nil
	}
	if t := strings.TrimSpace(cardigann.LoginCaptchaType(d)); t != "" {
		return fmt.Errorf("login captcha unsupported: %s", t)
	}
	return nil
}

func verifyLogin(ctx context.Context, d *cardigann.Definition, loginCookies map[string]string, do func(context.Context, fetch.Request) (fetch.Response, error)) error {
	path := cardigann.LoginTestPath(d)
	if path == "" || do == nil {
		return nil
	}
	fr := cardigann.FollowRedirect(d)
	req := fetch.Request{Method: "get", Base: d.BaseURL, Path: path, FollowRedirect: &fr}
	if raw := loginCookies["__raw__"]; raw != "" {
		req.Headers = map[string]string{"Cookie": raw}
	} else if len(loginCookies) > 0 {
		req.Headers = map[string]string{"Cookie": cookieHeader(loginCookies)}
	}
	resp, err := do(ctx, req)
	if err != nil {
		return fmt.Errorf("login test: %w", err)
	}
	sel := cardigann.LoginTestSelector(d)
	if sel == "" {
		return nil
	}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(resp.Body)))
	if err != nil {
		return fmt.Errorf("login test parse: %w", err)
	}
	if selectNodes(doc.Selection, sel).Length() == 0 {
		return fmt.Errorf("login test: %w", errLoginTestFailed)
	}
	return nil
}

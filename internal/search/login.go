package search

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/sergiobonfiglio/tomagnet/internal/cardigann"
	"github.com/sergiobonfiglio/tomagnet/internal/fetch"
)

func buildLoginRequest(d *cardigann.Definition, page, pageURL string) cardigann.RequestSpec {
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
	var form *goquery.Selection
	if formSel != "" {
		form = selectNodes(doc.Selection, formSel).First()
		if form.Length() > 0 {
			scope = form
			if submitPath == "" {
				if action, ok := form.Attr("action"); ok && strings.TrimSpace(action) != "" {
					spec.Path = resolveFormAction(pageURL, action)
				}
			}
		}
	}
	if submitPath != "" {
		spec.Path = submitPath
	}

	configuredInputs := spec.Inputs
	if cardigann.LoginUsesSelectors(d) {
		configuredInputs = resolveLoginInputNames(configuredInputs, scope, doc.Selection)
	}
	if form != nil && form.Length() > 0 {
		spec.Inputs = formInputs(form)
		maps.Copy(spec.Inputs, configuredInputs)
	} else {
		spec.Inputs = configuredInputs
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

func resolveLoginInputNames(inputs map[string]string, scope, document *goquery.Selection) map[string]string {
	resolved := map[string]string{}
	for selector, value := range inputs {
		q := selectNodes(scope, selector).First()
		if q.Length() == 0 {
			q = selectNodes(document, selector).First()
		}
		name, ok := q.Attr("name")
		if q.Length() == 0 || !ok || strings.TrimSpace(name) == "" {
			resolved[selector] = value
			continue
		}
		resolved[name] = value
	}
	return resolved
}

func resolveFormAction(pageURL, action string) string {
	base, err := url.Parse(pageURL)
	if err != nil || !base.IsAbs() {
		return action
	}
	reference, err := url.Parse(action)
	if err != nil {
		return action
	}
	return base.ResolveReference(reference).String()
}

func formInputs(form *goquery.Selection) map[string]string {
	inputs := map[string]string{}
	submitAdded := false
	form.Find("input, textarea, select, button").Each(func(_ int, field *goquery.Selection) {
		name := strings.TrimSpace(field.AttrOr("name", ""))
		if name == "" || field.Is("[disabled]") {
			return
		}

		tag := goquery.NodeName(field)
		typ := strings.ToLower(field.AttrOr("type", ""))
		switch tag {
		case "input":
			switch typ {
			case "checkbox", "radio":
				if !field.Is("[checked]") {
					return
				}
				inputs[name] = field.AttrOr("value", "on")
			case "submit":
				if submitAdded {
					return
				}
				submitAdded = true
				inputs[name] = field.AttrOr("value", "")
			case "button", "file", "image", "reset":
				return
			default:
				inputs[name] = field.AttrOr("value", "")
			}
		case "textarea":
			inputs[name] = field.Text()
		case "select":
			option := field.Find("option[selected]").First()
			if option.Length() == 0 {
				option = field.Find("option").First()
			}
			if option.Length() > 0 {
				inputs[name] = option.AttrOr("value", option.Text())
			}
		case "button":
			if typ != "" && typ != "submit" || submitAdded {
				return
			}
			submitAdded = true
			inputs[name] = field.AttrOr("value", field.Text())
		}
	})
	return inputs
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

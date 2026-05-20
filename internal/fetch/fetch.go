package fetch

import (
	"context"
	"crypto/sha1"
	"crypto/tls"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"
)

type Response struct {
	Body             []byte
	ContentType, URL string
	Cookies          map[string]string
}

type Request struct {
	Method         string
	Base           string
	Path           string
	Param          string
	Query          string
	Inputs         map[string]string
	Headers        map[string]string
	FollowRedirect *bool
}

type Session struct {
	timeout      time.Duration
	debug        func(string, ...any)
	jar          http.CookieJar
	certificates map[string]bool
}

func NewSession(timeout time.Duration, debug func(string, ...any), certificates ...string) *Session {
	jar, _ := cookiejar.New(nil)
	allowed := map[string]bool{}
	for _, c := range certificates {
		c = strings.ToLower(strings.TrimSpace(c))
		if c != "" {
			allowed[c] = true
		}
	}
	return &Session{timeout: timeout, debug: debug, jar: jar, certificates: allowed}
}

func Get(ctx context.Context, base, path, param, query string, timeout time.Duration, debug func(string, ...any)) (Response, error) {
	return Do(ctx, Request{Method: http.MethodGet, Base: base, Path: path, Param: param, Query: query}, timeout, debug)
}

func Do(ctx context.Context, spec Request, timeout time.Duration, debug func(string, ...any)) (Response, error) {
	return NewSession(timeout, debug).Do(ctx, spec)
}

func (s *Session) Do(ctx context.Context, spec Request) (Response, error) {
	u, err := buildURL(spec.Base, spec.Path)
	if err != nil {
		return Response{}, err
	}
	method := strings.ToUpper(spec.Method)
	if method == "" {
		method = http.MethodGet
	}
	vals := url.Values{}
	for k, v := range spec.Inputs {
		vals.Set(k, v)
	}
	if spec.Query != "" && spec.Param != "" && vals.Get(spec.Param) == "" {
		vals.Set(spec.Param, spec.Query)
	}
	var body io.Reader
	if method == http.MethodPost {
		body = strings.NewReader(vals.Encode())
	} else {
		q := u.Query()
		for k, vs := range vals {
			for _, v := range vs {
				q.Set(k, v)
			}
		}
		u.RawQuery = q.Encode()
	}
	if s != nil && s.debug != nil {
		s.debug("%s %s", method, u.String())
	}
	timeout := time.Duration(0)
	if s != nil {
		timeout = s.timeout
	}
	c := &http.Client{Timeout: timeout}
	if s != nil && s.jar != nil {
		c.Jar = s.jar
	}
	if s != nil && len(s.certificates) > 0 {
		c.Transport = &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true, VerifyConnection: func(cs tls.ConnectionState) error {
			if len(cs.PeerCertificates) == 0 {
				return nil
			}
			sum := sha1.Sum(cs.PeerCertificates[0].Raw)
			if s.certificates[hex.EncodeToString(sum[:])] {
				return nil
			}
			return errors.New("untrusted server certificate")
		}}}
	}
	if spec.FollowRedirect != nil && !*spec.FollowRedirect {
		c.CheckRedirect = func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }
	}
	req, err := http.NewRequestWithContext(ctx, method, u.String(), body)
	if err != nil {
		return Response{}, err
	}
	req.Header.Set("User-Agent", "tomagnet/0")
	if method == http.MethodPost {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	for k, v := range spec.Headers {
		req.Header.Set(k, v)
	}
	r, err := c.Do(req)
	if err != nil {
		return Response{}, err
	}
	defer r.Body.Close()
	b, err := io.ReadAll(r.Body)
	if err != nil {
		return Response{}, err
	}
	if r.StatusCode < 200 || r.StatusCode >= 300 {
		if !(spec.FollowRedirect != nil && !*spec.FollowRedirect && r.StatusCode >= 300 && r.StatusCode < 400) {
			return Response{}, &url.Error{Op: method, URL: u.String(), Err: http.ErrAbortHandler}
		}
	}
	cookies := map[string]string{}
	for _, c := range r.Cookies() {
		cookies[c.Name] = c.Value
	}
	return Response{Body: b, ContentType: r.Header.Get("Content-Type"), URL: r.Request.URL.String(), Cookies: cookies}, nil
}

func buildURL(base, path string) (*url.URL, error) {
	u, err := url.Parse(base)
	if err != nil {
		return nil, err
	}
	p, err := url.Parse(path)
	if err != nil {
		return nil, err
	}
	return u.ResolveReference(p), nil
}

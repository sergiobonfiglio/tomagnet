package normalize

import (
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func Abs(base, s string) string {
	if s == "" {
		return ""
	}
	u, err := url.Parse(s)
	if err != nil {
		return s
	}
	if u.IsAbs() {
		return s
	}
	b, err := url.Parse(base)
	if err != nil {
		return s
	}
	return b.ResolveReference(u).String()
}

func Size(s string) *int64 {
	s = strings.TrimSpace(strings.ReplaceAll(s, ",", ""))
	if s == "" {
		return nil
	}
	re := regexp.MustCompile(`(?i)([0-9]+(?:\.[0-9]+)?)\s*([kmgtp]?i?b|[kmgtp])?`)
	m := re.FindStringSubmatch(s)
	if m == nil {
		return nil
	}
	f, _ := strconv.ParseFloat(m[1], 64)
	mult := float64(1)
	switch strings.ToLower(m[2]) {
	case "k", "kb", "kib":
		mult = 1024
	case "m", "mb", "mib":
		mult = 1024 * 1024
	case "g", "gb", "gib":
		mult = 1024 * 1024 * 1024
	case "t", "tb", "tib":
		mult = 1024 * 1024 * 1024 * 1024
	}
	v := int64(f * mult)
	return &v
}
func Int(s string) *int {
	s = strings.TrimSpace(strings.ReplaceAll(s, ",", ""))
	i, err := strconv.Atoi(s)
	if err != nil {
		return nil
	}
	return &i
}
func Date(s string) *time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	if i, err := strconv.ParseInt(s, 10, 64); err == nil && i > 1000000000 {
		t := time.Unix(i, 0).UTC()
		return &t
	}
	layouts := []string{time.RFC3339, time.RFC1123Z, time.RFC1123, "2006-01-02 15:04:05", "2006-01-02", "02 Jan 2006", "Jan 2, 2006"}
	for _, l := range layouts {
		if t, err := time.Parse(l, s); err == nil {
			return &t
		}
	}
	return nil
}
func StrPtr(s string) *string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return &s
}

package output

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/sergiobonfiglio/tomagnet/internal/search"
)

func JSON(w io.Writer, r search.Response) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(r)
}
func Table(w io.Writer, r search.Response) error {
	fmt.Fprintln(w, "INDEXER\tTITLE\tSEED\tLEECH\tSIZE\tDATE\tMAGNET?\tTORRENT?")
	for _, x := range r.Results {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%t\t%t\n", x.Indexer, val(x.Title), ival(x.Seeders), ival(x.Leechers), sz(x.Size), date(x), x.MagnetURL != nil, x.DownloadURL != nil)
	}
	return nil
}
func val(p *string) string {
	if p == nil {
		return ""
	}
	s := strings.ReplaceAll(*p, "\t", " ")
	if len(s) > 80 {
		return s[:77] + "..."
	}
	return s
}
func ival(p *int) string {
	if p == nil {
		return ""
	}
	return fmt.Sprint(*p)
}
func sz(p *int64) string {
	if p == nil {
		return ""
	}
	return fmt.Sprint(*p)
}
func date(r search.Result) string {
	if r.PublishDate == nil {
		return ""
	}
	return r.PublishDate.Format("2006-01-02")
}

package testcmd

import (
	"context"
	"fmt"
	"io"

	"github.com/sergiobonfiglio/tomagnet/internal/config"
	"github.com/sergiobonfiglio/tomagnet/internal/search"
)

func Run(ctx context.Context, w io.Writer, indexers []config.Indexer, debug func(string, ...any)) bool {
	passed := 0
	for _, idx := range indexers {
		ok := false
		for _, q := range []string{"dune", "simpsons s01e01"} {
			r := search.Run(ctx, search.Options{Query: q, Indexers: []config.Indexer{idx}, Limit: 1, Concurrency: 1, Debug: debug})
			if len(r.Errors) == 0 {
				ok = true
				fmt.Fprintf(w, "PASS %s query=%q results=%d\n", idx.ID, q, len(r.Results))
				break
			}
		}
		if !ok {
			fmt.Fprintf(w, "FAIL %s\n", idx.ID)
		} else {
			passed++
		}
	}
	return passed > 0
}

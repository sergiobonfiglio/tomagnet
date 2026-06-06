# tomagnet

`tomagnet` is a Go CLI and library for querying torrent indexers and returning normalized magnet/search results.

It can fetch Cardigann/Jackett-style indexer definitions on demand, keep them in a local cache, and search one or more configured indexers with JSON or table output.

## Version

Current version: **0.3.6**

```bash
tomagnet --version
```

## Install

```bash
go install github.com/sergiobonfiglio/tomagnet/cmd/tomagnet@v0.3.6
```

Or build from a checkout:

```bash
go build -o tomagnet ./cmd/tomagnet
```

## Quick start

```bash
# Fetch indexer definitions into ./.tomagnet/definitions
tomagnet definitions sync

# Search the default public indexers
tomagnet search "dune" --output table

# Search a specific indexer, even if it is not in tomagnet.yaml
tomagnet search "dune" --indexer nyaasi --limit 20 --output json

# Smoke-test configured indexers
tomagnet test
tomagnet test nyaasi
```

## Configuration

`tomagnet.yaml` is optional. Without it, searches use the default public indexers: `btdig`, `yts`, `limetorrents`, and `thepiratebay`.

Create a local config from the example if you need custom settings:

```bash
cp tomagnet.example.yaml tomagnet.yaml
```

Example:

```yaml
default_timeout_seconds: 15
concurrency: 4
# Used by default/auto selection.
disabled_indexers:
  - thepiratebay
indexers:
  - id: "yts"
    # Optional. "auto" tries cached base URL first, then definition links.
    base_url: "auto"
  - id: "1337x"
    timeout_seconds: 10
    base_url: "https://1337x.to/"
```

Definitions resolve from `./definitions/<id>.yml|yaml`, then `./.tomagnet/definitions/<id>.yml|yaml`.

When `--indexer` is provided explicitly, tomagnet will use those ids even if they are not present in `tomagnet.yaml`. Matching config entries still provide overrides such as `base_url` and `timeout_seconds`.

## Library usage

The public Go API lives in `github.com/sergiobonfiglio/tomagnet/pkg/tomagnet`.

Main entry points:

- `Search(ctx, SearchOptions)` runs an intent-based search.
- `LoadDefinition(path)` loads a definition from disk.
- `LoadDefinitionByID(id)` resolves and loads a locally available definition.
- `SyncDefinitions()` refreshes the local public definitions cache.
- `DefaultIndexers()` returns tomagnet's default public indexers.

### Intent-based search

`Search` takes a high-level `Query` rather than transport-specific request
fields. Tomagnet inspects each definition, chooses the best supported mode, and
omits unset optional parameters automatically.

```go
package main

import (
    "context"
    "fmt"

    tomagnet "github.com/sergiobonfiglio/tomagnet/pkg/tomagnet"
)

func main() {
    definition, err := tomagnet.LoadDefinition("./.tomagnet/definitions/yts.yml")
    if err != nil {
        panic(err)
    }

    resp := tomagnet.Search(context.Background(), tomagnet.SearchOptions{
        Query: tomagnet.Query{
            Movie: &tomagnet.MovieQuery{
                Title:  "Dune",
                Year:   2021,
                IMDBID: "tt1160419",
            },
        },
        Indexers: []tomagnet.Indexer{{
            ID:         "yts",
            BaseURL:    "auto",
            Definition: definition,
        }},
        Limit: 20,
    })

    for _, err := range resp.Errors {
        fmt.Printf("indexer %s failed: %s\n", err.Indexer, err.Message)
    }
    for _, result := range resp.Results {
        fmt.Printf("%s -> %s\n", result.Title, result.MagnetURL)
    }
}
```

### Search types

- `Query.Text`: plain free-text search.
- `Query.Movie`: movie-specific intent with `Title`, `Year`, `IMDBID`, `TMDBID`.
- `Query.TV`: TV-specific intent with `Title`, `Year`, `Season`, `Episode`, and supported external ids.
- `SearchOptions.Categories`: optional category filter passed to supporting indexers.
- `SearchOptions.Indexers`: one or more indexers to query.
- `SearchOptions.Limit`: per-indexer result limit.
- `Response.Results`: normalized search hits.
- `Response.Errors`: per-indexer failures; one indexer can fail while others succeed.

### Definitions API

If you need to inspect or build definitions programmatically, `Definition` and
its exported nested structs mirror the YAML shape supported by the library.

## Development

```bash
go test ./...
```

Local/generated files are intentionally not committed:

- `tomagnet.yaml` for local config and credentials
- `.tomagnet/definitions/` for synced indexer definitions
- `tomagnet` and `bin/` for build artifacts
- `reports/` for generated reports

## License

MIT. See [LICENSE](LICENSE).

## Disclaimer

Use this tool only with content and indexers you are legally permitted to access. The project does not host or distribute torrent content.

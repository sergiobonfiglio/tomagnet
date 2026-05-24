# tomagnet

`tomagnet` is a Go CLI and library for querying torrent indexers and returning normalized magnet/search results.

It can fetch Cardigann/Jackett-style indexer definitions on demand, keep them in a local cache, and search one or more configured indexers with JSON or table output.

## Version

Current version: **0.1.0**

```bash
tomagnet --version
```

## Install

```bash
go install github.com/sergiobonfiglio/tomagnet/cmd/tomagnet@v0.1.0
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

# Search a specific configured indexer
tomagnet search "dune" --indexer nyaasi --limit 20 --output json

# Smoke-test configured indexers
tomagnet test
tomagnet test nyaasi
```

## Configuration

`tomagnet.yaml` is optional. Without it, searches use the default public indexers: `yts`, `1337x`, and `thepiratebay`.

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

## Library usage

```go
resp := tomagnet.Search(ctx, tomagnet.SearchOptions{
    Query: "dune",
    Indexers: []tomagnet.Indexer{{ID: "yts", BaseURL: "auto"}},
})
```

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

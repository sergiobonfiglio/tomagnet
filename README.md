# tomagnet

Independent Go module implementing `docs/tomagnet-plan.md` v0 CLI.

## Build

```bash
go build ./cmd/tomagnet
```

## Config

`tomagnet.yaml` is optional. Without it, searches use the default public indexers: `yts`, `1337x`, `thepiratebay`.

Optional config:

```yaml
default_timeout_seconds: 15
concurrency: 4
disabled_indexers:
  - thepiratebay
indexers:
  - id: yts
    base_url: auto
  - id: 1337x
    timeout_seconds: 10
    base_url: https://1337x.to/
```

## Commands

```bash
tomagnet definitions sync
tomagnet search "dune" --output json
tomagnet search "dune" --indexer nyaasi --limit 20 --output table
tomagnet test
tomagnet test nyaasi
```

Definitions resolve from `./definitions/<id>.yml|yaml`, then `./.tomagnet/definitions/<id>.yml|yaml`.

# Changelog

## 0.3.0

Breaking change for the Go library definition API.

- Replace the temporary `Definition.Raw map[string]any` API with typed public definition structs.
- Add `DecodeDefinition(io.Reader)` and `LoadDefinition(path)` so library consumers can reuse tomagnet's definition-file decoder.
- Keep `Definition *Definition` on `tomagnet.Indexer` for in-memory definitions.

## 0.2.0

Breaking change for the Go library API.

- Add public `tomagnet.Definition` for in-memory Cardigann/Jackett-style definitions.
- Add `Definition *Definition` to `tomagnet.Indexer` so library consumers can pass definitions without writing YAML files to tomagnet-owned paths.
- Internal search now prefers in-memory definitions and falls back to existing on-disk definition resolution when no in-memory definition is provided.

## 0.1.0

Initial public release.

- CLI for searching configured torrent indexers.
- JSON and table output.
- Local Cardigann/Jackett-style definition sync cache.
- Go library API under `pkg/tomagnet`.

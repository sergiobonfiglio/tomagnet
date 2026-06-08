# Changelog

## 0.3.8

Patch release.

- Add cache-dir-aware definition sync and load APIs for wrapper CLIs.

## 0.3.7

Patch release.

- Include tomagnet-bundled definitions such as `btdig` in `definitions sync`.
- Run `go fix ./...` to modernize standard-library usage.

## 0.3.6

Patch release.

- Add local `btdig` definition support.
- Use a `uTLS`-backed HTTPS transport with a Chrome-like ClientHello and HTTP/1.1 for indexers that reject the default Go client fingerprint.
- Fix `--indexer` so explicit ids override config selection and can be used even when not present in `tomagnet.yaml`.
- Add BTDig to the default public indexers list.
- Fix BTDig size parsing and remove an unreachable `return` in query rendering.

## 0.3.1

Patch release.

- Remove unused internal `ghItem` type.
- Remove an unused assignment in login request building.
- Verify with `staticcheck ./...`, `go test ./...`, and `go vet ./...`.

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

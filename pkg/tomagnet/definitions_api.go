package tomagnet

import internaldefinitions "github.com/sergiobonfiglio/tomagnet/internal/definitions"

// DefinitionsMetadata describes the result of a definitions sync.
type DefinitionsMetadata = internaldefinitions.Metadata

// SyncDefinitions refreshes the default local cache of public definitions.
func SyncDefinitions() (DefinitionsMetadata, error) {
	return internaldefinitions.Sync()
}

// SyncDefinitionsTo refreshes a caller-owned local cache of public definitions.
func SyncDefinitionsTo(cacheDir string) (DefinitionsMetadata, error) {
	return internaldefinitions.SyncTo(cacheDir)
}

// LoadDefinitionByID resolves and loads a definition by ID from local search
// paths such as ./definitions and ./.tomagnet/definitions.
func LoadDefinitionByID(id string) (*Definition, error) {
	return LoadDefinitionByIDFrom(id, internaldefinitions.CacheDir)
}

// LoadDefinitionByIDFrom resolves and loads a definition by ID from ./definitions
// and the provided definitions cache directory.
func LoadDefinitionByIDFrom(id, cacheDir string) (*Definition, error) {
	path, err := internaldefinitions.ResolveIn(id, cacheDir)
	if err != nil {
		return nil, err
	}
	return LoadDefinition(path)
}

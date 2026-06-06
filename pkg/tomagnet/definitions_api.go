package tomagnet

import internaldefinitions "github.com/sergiobonfiglio/tomagnet/internal/definitions"

// DefinitionsMetadata describes the result of a definitions sync.
type DefinitionsMetadata = internaldefinitions.Metadata

// SyncDefinitions refreshes the local cache of public definitions.
func SyncDefinitions() (DefinitionsMetadata, error) {
	return internaldefinitions.Sync()
}

// LoadDefinitionByID resolves and loads a definition by ID from local search
// paths such as ./definitions and ./.tomagnet/definitions.
func LoadDefinitionByID(id string) (*Definition, error) {
	path, err := internaldefinitions.Resolve(id)
	if err != nil {
		return nil, err
	}
	return LoadDefinition(path)
}

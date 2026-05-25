package tomagnet

import internaldefinitions "github.com/sergiobonfiglio/tomagnet/internal/definitions"

type DefinitionsMetadata = internaldefinitions.Metadata

func SyncDefinitions() (DefinitionsMetadata, error) {
	return internaldefinitions.Sync()
}

func LoadDefinitionByID(id string) (*Definition, error) {
	path, err := internaldefinitions.Resolve(id)
	if err != nil {
		return nil, err
	}
	return LoadDefinition(path)
}

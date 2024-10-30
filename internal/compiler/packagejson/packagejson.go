package packagejson

import (
	json2 "github.com/go-json-experiment/json"
)

type HeaderFields struct {
	Name    Expected[string] `json:"name"`
	Version Expected[string] `json:"version"`
	Type    Expected[string] `json:"type"`
}

type PathFields struct {
	TSConfig      Expected[string] `json:"tsconfig"`
	Main          Expected[string] `json:"main"`
	Types         Expected[string] `json:"types"`
	Typings       Expected[string] `json:"typings"`
	TypesVersions JSONValue        `json:"typesVersions"`
	Imports       JSONValue        `json:"imports"`
	Exports       Exports          `json:"exports"`
}

type DependencyFields struct {
	Dependencies         Expected[map[string]string] `json:"dependencies"`
	PeerDependencies     Expected[map[string]string] `json:"peerDependencies"`
	OptionalDependencies Expected[map[string]string] `json:"optionalDependencies"`
}

type Fields struct {
	HeaderFields
	PathFields
	DependencyFields
}

func Parse(data []byte) (*Fields, error) {
	var f Fields
	if err := json2.Unmarshal(data, &f); err != nil {
		return nil, err
	}
	return &f, nil
}

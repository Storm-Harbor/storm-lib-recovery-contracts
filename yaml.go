package recovery

import (
	"bytes"
	"encoding/json"
	"io"

	"go.yaml.in/yaml/v4"
)

const MaxDocumentBytes = 64 * 1024
const maxDocumentDepth = 32

// DecodeYAML decodes exactly one bounded, strict YAML configuration. Unknown or
// duplicate fields, anchors, aliases, custom tags and scalar type coercion are
// rejected. Errors contain no supplied document values or parser excerpts.
func DecodeYAML(data []byte, trusted Scope) (Configuration, error) {
	var zero Configuration
	if len(data) == 0 || len(data) > MaxDocumentBytes {
		return zero, violation(InvalidDocument, "document")
	}
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	var document yaml.Node
	if err := decoder.Decode(&document); err != nil {
		return zero, violation(InvalidDocument, "document")
	}
	var extra yaml.Node
	if err := decoder.Decode(&extra); err != io.EOF {
		return zero, violation(InvalidDocument, "document")
	}
	if !safeNode(&document, 0) {
		return zero, violation(InvalidDocument, "document")
	}
	// YAML field checking is case-sensitive, unlike encoding/json's struct
	// matching. The JSON pass below independently rejects scalar coercion.
	known := yaml.NewDecoder(bytes.NewReader(data))
	known.KnownFields(true)
	var shape Configuration
	if err := known.Decode(&shape); err != nil {
		return zero, violation(InvalidDocument, "document")
	}
	var raw any
	if err := document.Decode(&raw); err != nil {
		return zero, violation(InvalidDocument, "document")
	}
	encoded, err := json.Marshal(raw)
	if err != nil {
		return zero, violation(InvalidDocument, "document")
	}
	strict := json.NewDecoder(bytes.NewReader(encoded))
	strict.DisallowUnknownFields()
	var result Configuration
	if err := strict.Decode(&result); err != nil {
		return zero, violation(InvalidDocument, "document")
	}
	if err := result.Validate(trusted); err != nil {
		return zero, err
	}
	return result, nil
}

func safeNode(node *yaml.Node, depth int) bool {
	if depth > maxDocumentDepth || node.Anchor != "" {
		return false
	}
	switch node.Kind {
	case yaml.DocumentNode:
		if len(node.Content) != 1 {
			return false
		}
	case yaml.MappingNode:
		if node.Tag != "!!map" || len(node.Content)%2 != 0 {
			return false
		}
		seen := make(map[string]bool, len(node.Content)/2)
		for index := 0; index < len(node.Content); index += 2 {
			key := node.Content[index]
			if key.Kind != yaml.ScalarNode || key.Tag != "!!str" || seen[key.Value] {
				return false
			}
			seen[key.Value] = true
		}
	case yaml.SequenceNode:
		if node.Tag != "!!seq" {
			return false
		}
	case yaml.ScalarNode:
		return node.Tag == "!!str" || node.Tag == "!!int"
	default:
		return false
	}
	for _, child := range node.Content {
		if !safeNode(child, depth+1) {
			return false
		}
	}
	return true
}

package recovery

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"go.yaml.in/yaml/v4"
)

func TestRejectUntrustedYAMLWithoutDisclosingValues(t *testing.T) {
	valid := string(fixture(t, "exercise"))
	cases := map[string]string{
		"case-sensitive-field":   strings.Replace(valid, "apiVersion:", "ApiVersion:", 1),
		"empty":                  "",
		"whitespace":             " \n\t",
		"null":                   "null\n",
		"syntax":                 "password: [synthetic-credential\n",
		"unknown-root":           valid + "password: synthetic-credential\n",
		"unknown-vault-path":     strings.Replace(valid, "      purpose: restore-database", "      purpose: restore-database\n      vaultPath: synthetic-credential", 1),
		"inline-secret":          strings.Replace(valid, "      purpose: restore-database", "      purpose: restore-database\n      value: synthetic-credential", 1),
		"duplicate-root":         valid + "kind: RecoveryConfiguration\n",
		"duplicate-nested":       strings.Replace(valid, "    provisioning: create", "    provisioning: create\n    provisioning: existing", 1),
		"anchor":                 strings.Replace(valid, "name: service-exercise", "name: &secret synthetic-credential", 1),
		"alias":                  valid + "password: *missing\n",
		"merge":                  strings.Replace(valid, "metadata:\n", "metadata:\n  <<: {name: synthetic-credential}\n", 1),
		"custom-tag":             strings.Replace(valid, "name: service-exercise", "name: !secret synthetic-credential", 1),
		"numeric-string":         strings.Replace(valid, "organizationId: org-example", "organizationId: 123", 1),
		"boolean":                strings.Replace(valid, "name: service-exercise", "name: true", 1),
		"null-field":             strings.Replace(valid, "projectId: project-example", "projectId: null", 1),
		"negative-expiry":        strings.Replace(valid, "ttlSeconds: 3600", "ttlSeconds: -1", 1),
		"fractional-expiry":      strings.Replace(valid, "ttlSeconds: 3600", "ttlSeconds: 1.5", 1),
		"overflow-expiry":        strings.Replace(valid, "ttlSeconds: 3600", "ttlSeconds: 4294967296", 1),
		"string-expiry":          strings.Replace(valid, "ttlSeconds: 3600", "ttlSeconds: '3600'", 1),
		"nonfinite-expiry":       strings.Replace(valid, "ttlSeconds: 3600", "ttlSeconds: .inf", 1),
		"second-document":        valid + "---\npassword: synthetic-credential\n",
		"second-empty-document":  valid + "---\n",
		"second-broken-document": valid + "---\n[synthetic-credential\n",
		"nonstring-key":          "1: synthetic-credential\n",
		"sequence-root":          "- synthetic-credential\n",
		"oversized":              strings.Repeat("#", MaxDocumentBytes+1),
		"invalid-utf8":           string([]byte{0xff, 0, 0xfe}),
		"deep":                   strings.Repeat("[", 40) + "value" + strings.Repeat("]", 40),
	}
	for name, data := range cases {
		t.Run(name, func(t *testing.T) {
			result, err := DecodeYAML([]byte(data), exampleScope)
			assertViolation(t, err, InvalidDocument, "document")
			if result.APIVersion != "" {
				t.Fatal("invalid input returned partial configuration")
			}
			encoded, marshalErr := json.Marshal(err)
			if marshalErr != nil {
				t.Fatal(marshalErr)
			}
			if strings.Contains(string(encoded), "synthetic-credential") || strings.Contains(err.Error(), "synthetic-credential") {
				t.Fatal("error disclosed supplied material")
			}
		})
	}
}

func TestDecodedSemanticViolationRemainsTyped(t *testing.T) {
	data := strings.Replace(string(fixture(t, "exercise")), "purpose: exercise", "purpose: invalid", 1)
	_, err := DecodeYAML([]byte(data), exampleScope)
	assertViolation(t, err, InvalidConfiguration, "spec.purpose")
	var violation *Violation
	if !errors.As(err, &violation) {
		t.Fatal(err)
	}
}

func TestYAMLAllowsCRLFAndTrailingComments(t *testing.T) {
	data := strings.ReplaceAll(string(fixture(t, "exercise")), "\n", "\r\n") + "\r\n# no additional document\r\n"
	if _, err := DecodeYAML([]byte(data), exampleScope); err != nil {
		t.Fatal(err)
	}
}

func TestASTSafetyRejectsInvalidStructuralStates(t *testing.T) {
	cases := []*yaml.Node{
		{Kind: yaml.DocumentNode},
		{Kind: yaml.MappingNode, Tag: "!custom"},
		{Kind: yaml.MappingNode, Tag: "!!map", Content: []*yaml.Node{{Kind: yaml.ScalarNode, Tag: "!!str", Value: "odd"}}},
		{Kind: yaml.SequenceNode, Tag: "!custom"},
		{Kind: yaml.AliasNode},
		{Kind: yaml.ScalarNode, Tag: "!!str", Anchor: "unexpected"},
		{Kind: yaml.MappingNode, Tag: "!!map", Content: []*yaml.Node{{Kind: yaml.SequenceNode, Tag: "!!seq"}, {Kind: yaml.ScalarNode, Tag: "!!str"}}},
	}
	for _, node := range cases {
		if safeNode(node, 0) {
			t.Fatalf("accepted unsafe node kind %d", node.Kind)
		}
	}
}

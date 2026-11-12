package recovery

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

func TestPublishedSchemaMatchesConfigurationBoundary(t *testing.T) {
	data, err := os.ReadFile("schemas/recovery-configuration.schema.json")
	if err != nil {
		t.Fatal(err)
	}
	var document any
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}
	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource("schema.json", document); err != nil {
		t.Fatal(err)
	}
	schema, err := compiler.Compile("schema.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, purpose := range []string{"exercise", "restore"} {
		value := configuration(t, purpose)
		for _, provisioning := range []Provisioning{CreateTarget, ExistingTarget} {
			value.Spec.Target.Provisioning = provisioning
			encoded, err := json.Marshal(value)
			if err != nil {
				t.Fatal(err)
			}
			var raw any
			if err := json.Unmarshal(encoded, &raw); err != nil {
				t.Fatal(err)
			}
			if err := schema.Validate(raw); err != nil {
				t.Fatalf("%s/%s: %v", purpose, provisioning, err)
			}
		}
	}
	base, err := json.Marshal(configuration(t, "exercise"))
	if err != nil {
		t.Fatal(err)
	}
	for name, invalid := range map[string]string{
		"inline-secret":         strings.Replace(string(base), `"purpose":"restore-database"`, `"purpose":"restore-database","value":"synthetic-credential"`, 1),
		"wrong-scope-shape":     strings.Replace(string(base), `"organizationId":"org-example"`, `"organizationId":"bad/scope"`, 1),
		"wrong-lifecycle":       strings.Replace(string(base), `"lifecycle":"ephemeral"`, `"lifecycle":"retained"`, 1),
		"wrong-cleanup":         strings.Replace(string(base), `"cleanup":"run-owned"`, `"cleanup":"retain"`, 1),
		"missing-expiry":        strings.Replace(string(base), `,"ttlSeconds":3600`, ``, 1),
		"zero-expiry":           strings.Replace(string(base), `"ttlSeconds":3600`, `"ttlSeconds":0`, 1),
		"string-expiry":         strings.Replace(string(base), `"ttlSeconds":3600`, `"ttlSeconds":"3600"`, 1),
		"overflow-expiry":       strings.Replace(string(base), `"ttlSeconds":3600`, `"ttlSeconds":4294967296`, 1),
		"unknown-provider-path": strings.Replace(string(base), `"providerRef":"gcp-recovery"`, `"providerRef":"gcp-recovery","vaultPath":"synthetic-path"`, 1),
	} {
		t.Run(name, func(t *testing.T) {
			var raw any
			if err := json.Unmarshal([]byte(invalid), &raw); err != nil {
				t.Fatal(err)
			}
			if err := schema.Validate(raw); err == nil {
				t.Fatal("schema accepted invalid boundary")
			}
			if _, err := DecodeYAML([]byte(invalid), exampleScope); err == nil {
				t.Fatal("decoder disagrees with schema")
			}
		})
	}
}

package recovery

import (
	"encoding/json"
	"errors"
	"os"
	"strings"
	"sync"
	"testing"
)

var exampleScope = Scope{OrganizationID: "org-example", ProjectID: "project-example"}

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile("testdata/" + name + ".yaml")
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func configuration(t *testing.T, name string) Configuration {
	t.Helper()
	value, err := DecodeYAML(fixture(t, name), exampleScope)
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func assertViolation(t *testing.T, err error, code ErrorCode, field string) {
	t.Helper()
	var result *Violation
	if !errors.As(err, &result) {
		t.Fatalf("expected typed violation, got %T: %v", err, err)
	}
	if result.Code != code || result.Arguments["field"] != field {
		t.Fatalf("unexpected violation: %#v", result)
	}
	if result.Module != "recovery-contracts" || result.Key != "contracts."+string(code) || result.Error() != string(code) {
		t.Fatalf("unstable public error contract: %#v", result)
	}
	wantCategory := "validation"
	if code == ScopeMismatch {
		wantCategory = "authorization"
	}
	if result.Category != wantCategory {
		t.Fatalf("category: %s", result.Category)
	}
}

func TestValidPurposesAndProvisioning(t *testing.T) {
	for _, purpose := range []string{"exercise", "restore"} {
		for _, provisioning := range []Provisioning{CreateTarget, ExistingTarget} {
			t.Run(purpose+"/"+string(provisioning), func(t *testing.T) {
				c := configuration(t, purpose)
				c.Spec.Target.Provisioning = provisioning
				if err := c.Validate(exampleScope); err != nil {
					t.Fatal(err)
				}
			})
		}
	}
	if err := exampleScope.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestConfigurationRejectsUnsafeIntent(t *testing.T) {
	cases := []struct {
		name, fixture, field string
		code                 ErrorCode
		mutate               func(*Configuration)
	}{
		{"version", "exercise", "apiVersion", InvalidConfiguration, func(c *Configuration) { c.APIVersion = "other/v1" }},
		{"kind", "exercise", "kind", InvalidConfiguration, func(c *Configuration) { c.Kind = "Other" }},
		{"name", "exercise", "metadata.name", InvalidConfiguration, func(c *Configuration) { c.Metadata.Name = "bad/name" }},
		{"missing-org", "exercise", "metadata.scope.organizationId", InvalidScope, func(c *Configuration) { c.Metadata.Scope.OrganizationID = "" }},
		{"missing-project", "exercise", "metadata.scope.projectId", InvalidScope, func(c *Configuration) { c.Metadata.Scope.ProjectID = "" }},
		{"foreign-org", "exercise", "metadata.scope", ScopeMismatch, func(c *Configuration) { c.Metadata.Scope.OrganizationID = "other" }},
		{"foreign-project", "exercise", "metadata.scope", ScopeMismatch, func(c *Configuration) { c.Metadata.Scope.ProjectID = "other" }},
		{"purpose", "exercise", "spec.purpose", InvalidConfiguration, func(c *Configuration) { c.Spec.Purpose = "unknown" }},
		{"source-name", "exercise", "spec.source.name", InvalidConfiguration, func(c *Configuration) { c.Spec.Source.Name = "" }},
		{"source-invalid-scope", "exercise", "spec.source.scope.projectId", InvalidScope, func(c *Configuration) { c.Spec.Source.Scope.ProjectID = "" }},
		{"source-foreign", "exercise", "spec.source.scope", ScopeMismatch, func(c *Configuration) { c.Spec.Source.Scope.OrganizationID = "other" }},
		{"target-name", "exercise", "spec.target.name", InvalidConfiguration, func(c *Configuration) { c.Spec.Target.Name = "" }},
		{"provider-ref", "exercise", "spec.target.providerRef", InvalidConfiguration, func(c *Configuration) { c.Spec.Target.ProviderRef = "https://provider" }},
		{"placement-ref", "exercise", "spec.target.placementRef", InvalidConfiguration, func(c *Configuration) { c.Spec.Target.PlacementRef = "" }},
		{"provisioning", "exercise", "spec.target.provisioning", InvalidConfiguration, func(c *Configuration) { c.Spec.Target.Provisioning = "unknown" }},
		{"exercise-lifecycle", "exercise", "spec.target.lifecycle", InvalidConfiguration, func(c *Configuration) { c.Spec.Target.Lifecycle = Retained }},
		{"exercise-cleanup", "exercise", "spec.target.cleanup", InvalidConfiguration, func(c *Configuration) { c.Spec.Target.Cleanup = Retain }},
		{"exercise-expiry", "exercise", "spec.target.ttlSeconds", InvalidConfiguration, func(c *Configuration) { c.Spec.Target.TTLSeconds = 0 }},
		{"restore-lifecycle", "restore", "spec.target.lifecycle", InvalidConfiguration, func(c *Configuration) { c.Spec.Target.Lifecycle = Ephemeral }},
		{"restore-cleanup", "restore", "spec.target.cleanup", InvalidConfiguration, func(c *Configuration) { c.Spec.Target.Cleanup = RunOwned }},
		{"restore-expiry", "restore", "spec.target.ttlSeconds", InvalidConfiguration, func(c *Configuration) { c.Spec.Target.TTLSeconds = 1 }},
		{"source-target-alias", "exercise", "spec.target.name", SourceTargetConflict, func(c *Configuration) { c.Spec.Target.Name = c.Spec.Source.Name }},
		{"secret-name", "exercise", "spec.secrets.name", InvalidConfiguration, func(c *Configuration) { c.Spec.Secrets[0].Name = "" }},
		{"secret-purpose", "exercise", "spec.secrets.purpose", InvalidConfiguration, func(c *Configuration) { c.Spec.Secrets[0].Purpose = "" }},
		{"secret-invalid-scope", "exercise", "spec.secrets.scope.projectId", InvalidScope, func(c *Configuration) { c.Spec.Secrets[0].Scope.ProjectID = "" }},
		{"secret-foreign", "exercise", "spec.secrets.scope", ScopeMismatch, func(c *Configuration) { c.Spec.Secrets[0].Scope.ProjectID = "other" }},
		{"secret-duplicate", "exercise", "spec.secrets.name", DuplicateSecretReference, func(c *Configuration) { c.Spec.Secrets = append(c.Spec.Secrets, c.Spec.Secrets[0]) }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := configuration(t, tc.fixture)
			tc.mutate(&c)
			assertViolation(t, c.Validate(exampleScope), tc.code, tc.field)
		})
	}
}

func TestTrustedScopeMustBeIndependentAndWellFormed(t *testing.T) {
	c := configuration(t, "exercise")
	assertViolation(t, c.Validate(Scope{}), InvalidScope, "trustedScope.organizationId")
	assertViolation(t, c.Validate(Scope{OrganizationID: "org-example"}), InvalidScope, "trustedScope.projectId")
	assertViolation(t, Scope{OrganizationID: "bad\nvalue", ProjectID: "ok"}.Validate(), InvalidScope, "scope.organizationId")
	assertViolation(t, Scope{OrganizationID: "ok", ProjectID: strings.Repeat("a", 129)}.Validate(), InvalidScope, "scope.projectId")
	if err := (Scope{OrganizationID: strings.Repeat("a", 128), ProjectID: "123._:-"}).Validate(); err != nil {
		t.Fatal(err)
	}
	c.Metadata.Name = strings.Repeat("a", 63)
	if err := c.Validate(exampleScope); err != nil {
		t.Fatal(err)
	}
	c.Metadata.Name += "a"
	assertViolation(t, c.Validate(exampleScope), InvalidConfiguration, "metadata.name")
}

func TestValidationDoesNotMutateConfiguration(t *testing.T) {
	c := configuration(t, "exercise")
	before, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Validate(exampleScope); err != nil {
		t.Fatal(err)
	}
	after, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("validation mutated input")
	}
}

func TestConcurrentDecodingPreservesScopeIsolation(t *testing.T) {
	data := fixture(t, "exercise")
	var workers sync.WaitGroup
	for index := 0; index < 30; index++ {
		workers.Add(1)
		go func(index int) {
			defer workers.Done()
			scope := exampleScope
			if index%2 != 0 {
				scope.ProjectID = "other"
			}
			result, err := DecodeYAML(data, scope)
			if index%2 == 0 {
				if err != nil || result.Metadata.Scope != exampleScope {
					t.Errorf("valid caller: %v", err)
				}
			} else {
				assertViolation(t, err, ScopeMismatch, "metadata.scope")
				if result.APIVersion != "" {
					t.Error("rejected scope received a partial result")
				}
			}
		}(index)
	}
	workers.Wait()
}

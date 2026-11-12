package recovery

import "regexp"

const (
	APIVersion        = "recovery.stormharbor.io/v1alpha1"
	ConfigurationKind = "RecoveryConfiguration"
)

var (
	identifierPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$`)
	namePattern       = regexp.MustCompile(`^[a-z](?:[a-z0-9-]{0,61}[a-z0-9])?$`)
)

// Scope identifies product authorization boundaries, not cloud accounts.
type Scope struct {
	OrganizationID string `json:"organizationId" yaml:"organizationId"`
	ProjectID      string `json:"projectId" yaml:"projectId"`
}

// Validate checks shape only. Authentication and membership are caller duties.
func (s Scope) Validate() error { return validateScope(s, "scope") }

func validateScope(s Scope, field string) error {
	if !identifierPattern.MatchString(s.OrganizationID) {
		return violation(InvalidScope, field+".organizationId")
	}
	if !identifierPattern.MatchString(s.ProjectID) {
		return violation(InvalidScope, field+".projectId")
	}
	return nil
}

func requireScope(actual, trusted Scope, field string) error {
	if err := validateScope(actual, field); err != nil {
		return err
	}
	if actual != trusted {
		return violation(ScopeMismatch, field)
	}
	return nil
}

type Metadata struct {
	Name  string `json:"name" yaml:"name"`
	Scope Scope  `json:"scope" yaml:"scope"`
}

// ResourceReference is a logical alias resolved under the caller's scope.
type ResourceReference struct {
	Name  string `json:"name" yaml:"name"`
	Scope Scope  `json:"scope" yaml:"scope"`
}

// SecretReference contains an alias and purpose, never a value or vault path.
type SecretReference struct {
	Name    string `json:"name" yaml:"name"`
	Scope   Scope  `json:"scope" yaml:"scope"`
	Purpose string `json:"purpose" yaml:"purpose"`
}

// Purpose distinguishes an exercise from an intended retained restoration.
type Purpose string

const (
	Exercise Purpose = "exercise"
	Restore  Purpose = "restore"
)

type Provisioning string

const (
	CreateTarget   Provisioning = "create"
	ExistingTarget Provisioning = "existing"
)

type Lifecycle string

const (
	Ephemeral Lifecycle = "ephemeral"
	Retained  Lifecycle = "retained"
)

type Cleanup string

const (
	RunOwned Cleanup = "run-owned"
	Retain   Cleanup = "retain"
)

// Target declares intent. Provider/placement references are opaque aliases.
// RunOwned permits cleanup only after runtime ownership is independently proven;
// it never authorizes deletion of a pre-existing project or unrelated assets.
type Target struct {
	Name         string       `json:"name" yaml:"name"`
	ProviderRef  string       `json:"providerRef" yaml:"providerRef"`
	Provisioning Provisioning `json:"provisioning" yaml:"provisioning"`
	PlacementRef string       `json:"placementRef" yaml:"placementRef"`
	Lifecycle    Lifecycle    `json:"lifecycle" yaml:"lifecycle"`
	Cleanup      Cleanup      `json:"cleanup" yaml:"cleanup"`
	TTLSeconds   uint32       `json:"ttlSeconds,omitempty" yaml:"ttlSeconds,omitempty"`
}

type Specification struct {
	Purpose Purpose           `json:"purpose" yaml:"purpose"`
	Source  ResourceReference `json:"source" yaml:"source"`
	Target  Target            `json:"target" yaml:"target"`
	Secrets []SecretReference `json:"secrets,omitempty" yaml:"secrets,omitempty"`
}

// Configuration is a Storm Harbor resource, not a native Kubernetes object.
type Configuration struct {
	APIVersion string        `json:"apiVersion" yaml:"apiVersion"`
	Kind       string        `json:"kind" yaml:"kind"`
	Metadata   Metadata      `json:"metadata" yaml:"metadata"`
	Spec       Specification `json:"spec" yaml:"spec"`
}

// Validate checks a configuration against a scope independently established by
// the caller. Supplying the YAML's own scope as trusted bypasses that boundary;
// identity/authorization adapters must establish trusted first.
func (c Configuration) Validate(trusted Scope) error {
	if err := validateScope(trusted, "trustedScope"); err != nil {
		return err
	}
	if c.APIVersion != APIVersion {
		return violation(InvalidConfiguration, "apiVersion")
	}
	if c.Kind != ConfigurationKind {
		return violation(InvalidConfiguration, "kind")
	}
	if !namePattern.MatchString(c.Metadata.Name) {
		return violation(InvalidConfiguration, "metadata.name")
	}
	if err := requireScope(c.Metadata.Scope, trusted, "metadata.scope"); err != nil {
		return err
	}
	if c.Spec.Purpose != Exercise && c.Spec.Purpose != Restore {
		return violation(InvalidConfiguration, "spec.purpose")
	}
	if !namePattern.MatchString(c.Spec.Source.Name) {
		return violation(InvalidConfiguration, "spec.source.name")
	}
	if err := requireScope(c.Spec.Source.Scope, trusted, "spec.source.scope"); err != nil {
		return err
	}
	if err := validateTarget(c.Spec.Target, c.Spec.Purpose); err != nil {
		return err
	}
	if c.Spec.Source.Name == c.Spec.Target.Name {
		return violation(SourceTargetConflict, "spec.target.name")
	}
	seen := make(map[string]bool, len(c.Spec.Secrets))
	for _, ref := range c.Spec.Secrets {
		if !namePattern.MatchString(ref.Name) {
			return violation(InvalidConfiguration, "spec.secrets.name")
		}
		if !namePattern.MatchString(ref.Purpose) {
			return violation(InvalidConfiguration, "spec.secrets.purpose")
		}
		if err := requireScope(ref.Scope, trusted, "spec.secrets.scope"); err != nil {
			return err
		}
		if seen[ref.Name] {
			return violation(DuplicateSecretReference, "spec.secrets.name")
		}
		seen[ref.Name] = true
	}
	return nil
}

func validateTarget(t Target, purpose Purpose) error {
	for _, entry := range []struct{ value, field string }{
		{t.Name, "spec.target.name"}, {t.ProviderRef, "spec.target.providerRef"}, {t.PlacementRef, "spec.target.placementRef"},
	} {
		if !namePattern.MatchString(entry.value) {
			return violation(InvalidConfiguration, entry.field)
		}
	}
	if t.Provisioning != CreateTarget && t.Provisioning != ExistingTarget {
		return violation(InvalidConfiguration, "spec.target.provisioning")
	}
	if purpose == Exercise {
		if t.Lifecycle != Ephemeral {
			return violation(InvalidConfiguration, "spec.target.lifecycle")
		}
		if t.Cleanup != RunOwned {
			return violation(InvalidConfiguration, "spec.target.cleanup")
		}
		if t.TTLSeconds == 0 {
			return violation(InvalidConfiguration, "spec.target.ttlSeconds")
		}
	} else {
		if t.Lifecycle != Retained {
			return violation(InvalidConfiguration, "spec.target.lifecycle")
		}
		if t.Cleanup != Retain {
			return violation(InvalidConfiguration, "spec.target.cleanup")
		}
		if t.TTLSeconds != 0 {
			return violation(InvalidConfiguration, "spec.target.ttlSeconds")
		}
	}
	return nil
}

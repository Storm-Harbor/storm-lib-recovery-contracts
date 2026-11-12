# storm-lib-recovery-contracts

Versioned Go contracts and strict YAML validation for scoped recovery configuration, logical secret references and recovery targets. Runtime baseline: Go 1.26.x, pinned to 1.26.11 for this candidate.

This initial producer validates configuration offline. It performs no authentication, cloud mutation, secret retrieval, GitHub API call or recovery execution. GitHub App onboarding, policy evaluation, a CLI, adapters, CRDs and the journey engine are separate future deliveries.

## Contract

`recovery.stormharbor.io/v1alpha1`, kind `RecoveryConfiguration`, is a new provider-neutral contract. It does not replace the control-plane repository's legacy `recovery.controlplane.io` GCP simulation contract and is not an installed Kubernetes CRD.

- [Exercise fixture](testdata/exercise.yaml): create an ephemeral target, request run-owned cleanup and declare an expiry.
- [Restore fixture](testdata/restore.yaml): retain an existing target, without automatic expiry or cleanup.
- [JSON Schema](schemas/recovery-configuration.schema.json): strict structure and lifecycle validation. Its `$id` is a logical identifier, not a promise of an HTTP schema endpoint; consume the file at an immutable module version.
- [Adoption documentation](https://storm-harbor.github.io/storm-harbor-control-plane/): technical guides and implementation status.

### Validation in a caller

```go
import recovery "github.com/Storm-Harbor/storm-lib-recovery-contracts"

// Establish these IDs from authenticated identity, membership and policy.
// Never construct trustedScope by copying the document being validated.
trustedScope := recovery.Scope{
    OrganizationID: authorizedOrganizationID,
    ProjectID: authorizedProjectID,
}
configuration, err := recovery.DecodeYAML(documentBytes, trustedScope)
if err != nil {
    // Map the typed violation to a safe outer response and localized text.
    return err
}
_ = configuration // Still requires authorization, planning and execution gates.
```

The public error exposes stable code, category, module, localization key and a safe field path; it never includes YAML values or parser excerpts. The ten `i18n` catalogs belong to outer presentation adapters. The library does not select a user locale.

`DecodeYAML` accepts one document up to 64 KiB and depth 32. It rejects unknown/duplicate/case-mismatched fields, anchors, aliases, custom tags, multiple documents and scalar coercion. Secret values and vault paths have no accepted field. Successful `Validate` checks a Go value; use `DecodeYAML` at an untrusted YAML boundary to enforce the parser restrictions too.

### Security and lifecycle boundaries

Scopes must match the caller's independently established scope, including source and secret references. This check cannot authenticate the caller, detect revoked membership or protect the caller that supplies a forged trusted scope. Such enforcement belongs to future authenticated adapters.

Provider and placement fields are scoped logical aliases. Provider accounts, GCP folder/project/billing bindings, actual regions, network rules, workload identity and vault paths are resolved by future customer-controlled adapters. A name such as `gcp-recovery` or `aws-recovery` does not prove an implemented provider capability.

Exercises require `ephemeral`, `run-owned` and a positive `ttlSeconds`. They may create a target or use an existing sandbox; cleanup may remove only resources independently proven to have been created by that run. Existing projects and unrelated assets remain outside cleanup. Expiry is a request for a future authorized janitor, not a running deletion timer in this library. Restores require `retained` and `retain`, with omitted/zero expiry.

The validator rejects equal source/target aliases. Different aliases may still resolve to the same real resource: future adapters must compare actual identities, independently protect source resources, validate permissions/capabilities and enforce approvals before any mutation. Installation policies can be stricter than this structural contract. JSON Schema alone cannot check trusted-scope equality, unique secret aliases or source/target identity; use Go validation and runtime checks together.

## Development and delivery

Run `make check` with the pinned Go toolchain. It checks formatting, vet, a single combined race/statement-coverage suite, build and module checksums. Coverage threshold is 97% of statements; Go's native profile also reports function statement coverage. It does not measure branch or line coverage independently. JSON Schema conformance is tested with a test-only dependency; no live infrastructure is needed.

The initial implementation is a candidate tracked in [issue #1](https://github.com/Storm-Harbor/storm-lib-recovery-contracts/issues/1). Consumers wait for the user's manual PR merge and an immutable release before importing it. An alpha schema may change incompatibly only through an explicitly versioned API/migration; do not silently replace the legacy simulation contract. No release version is claimed by this README.

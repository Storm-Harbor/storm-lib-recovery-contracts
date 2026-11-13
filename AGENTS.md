# Recovery contracts

Read the Storm Harbor shared AGENTS.md and specifications. In the full workspace they are at `../../.github/AGENTS.md`; the published owner is https://github.com/Storm-Harbor/.github.

This module owns shared recovery configuration semantics, native typed validation errors, logical references and provider-independent target intent. It uses Go 1.26.x. It does not authenticate callers, resolve secrets, authorize a provider or execute recovery. A trusted caller supplies the expected scope independently of YAML.

Preserve the existing mock configuration in the documentation repository. The resource defined here is a separate alpha contract, not an installed Kubernetes CRD. Public examples and tests are synthetic. Never copy private adopter identifiers, vault paths, credentials or Terraform state into this repository.

Use the owning issue and Project #1. The default/integration branch is `develop`; production is `master`. New issue branches start from fetched/updated `master` and target `develop`. Only a separate same-repository `develop` PR may target `master`. Issue #3 owns branch-governance alignment; preserve the immutable release from issue #1. The user merges PRs manually. Publish an immutable module version only after the producer merge and checks, before implementing consumers.

Use `go test -race -count=1 -covermode=atomic -coverpkg=./... -coverprofile=.artifacts/coverage.out ./...` once per equivalent validation run, plus gofmt, vet, build and module verification. Report measured statement/function coverage without inventing an independent branch or line measurement. Libraries require at least 97% measured statement coverage. Keep the ten error catalogs and placeholder sets consistent.

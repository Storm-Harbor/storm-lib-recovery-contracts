// Package recovery defines the alpha contract for scoped recovery configuration.
// Validation is local and deterministic: it performs no network, filesystem,
// credential-resolution or infrastructure operations. Successful validation
// never establishes the caller's identity or grants execution authority.
package recovery

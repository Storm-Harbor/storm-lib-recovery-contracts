package recovery

// ErrorCode is a stable machine-readable validation outcome.
type ErrorCode string

const (
	InvalidDocument          ErrorCode = "INVALID_DOCUMENT"
	InvalidConfiguration     ErrorCode = "INVALID_CONFIGURATION"
	InvalidScope             ErrorCode = "INVALID_SCOPE"
	ScopeMismatch            ErrorCode = "SCOPE_MISMATCH"
	SourceTargetConflict     ErrorCode = "SOURCE_TARGET_CONFLICT"
	DuplicateSecretReference ErrorCode = "DUPLICATE_SECRET_REFERENCE"
)

// Violation exposes only stable codes, localization keys and safe field paths.
// Raw parser errors and supplied values are deliberately not retained.
type Violation struct {
	Code      ErrorCode         `json:"code"`
	Module    string            `json:"module"`
	Category  string            `json:"category"`
	Key       string            `json:"errorKey"`
	Arguments map[string]string `json:"arguments"`
}

// Error returns a machine code. Presentation adapters resolve Key and Arguments.
func (v *Violation) Error() string { return string(v.Code) }

func violation(code ErrorCode, field string) *Violation {
	category := "validation"
	if code == ScopeMismatch {
		category = "authorization"
	}
	return &Violation{
		Code: code, Module: "recovery-contracts", Category: category,
		Key:       "contracts." + string(code),
		Arguments: map[string]string{"field": field},
	}
}

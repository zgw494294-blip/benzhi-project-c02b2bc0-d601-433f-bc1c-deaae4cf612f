package domain

import "fmt"

type ErrorCode string

const (
	CodeValidation           ErrorCode = "validation_error"
	CodeTargetConflict       ErrorCode = "target_conflict"
	CodeBaselineNotReady     ErrorCode = "baseline_not_ready"
	CodeEmptyRevision        ErrorCode = "empty_revision"
	CodeRemediationReference ErrorCode = "remediation_reference_invalid"
	CodeMonitoringInterlock  ErrorCode = "monitoring_interlock_failed"
	CodeReviewReasons        ErrorCode = "review_reasons_required"
	CodeReviewUnresolved     ErrorCode = "review_requirements_unresolved"
	CodePermitPrecheck       ErrorCode = "permit_precheck_failed"
	CodeNotFound             ErrorCode = "not_found"
	CodeConflict             ErrorCode = "version_conflict"
	CodeState                ErrorCode = "invalid_state"
	CodeForbidden            ErrorCode = "forbidden"
	CodeFrozen               ErrorCode = "case_frozen"
	CodeCorrupt              ErrorCode = "data_corrupt"
)

type Error struct {
	Code    ErrorCode
	Message string
	Details any
}

func (e *Error) Error() string { return e.Message }
func NewError(code ErrorCode, format string, args ...any) error {
	return &Error{Code: code, Message: fmt.Sprintf(format, args...)}
}
func NewDetailedError(code ErrorCode, details any, format string, args ...any) error {
	return &Error{Code: code, Message: fmt.Sprintf(format, args...), Details: details}
}
func ErrorCodeOf(err error) ErrorCode {
	if e, ok := err.(*Error); ok {
		return e.Code
	}
	return ""
}

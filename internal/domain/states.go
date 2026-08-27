package domain

type CaseState string

const (
	StateDraft               CaseState = "draft"
	StateBaselineReady       CaseState = "baseline_ready"
	StateRemediationRequired CaseState = "remediation_required"
	StateReviewReady         CaseState = "review_ready"
	StateChangesRequired     CaseState = "changes_required"
	StateApproved            CaseState = "approved"
	StateFrozen              CaseState = "frozen"
)

func (s CaseState) Mutable() bool { return s != StateFrozen }
func (s CaseState) CanAcceptRevision() bool {
	return s == StateBaselineReady || s == StateRemediationRequired || s == StateChangesRequired
}

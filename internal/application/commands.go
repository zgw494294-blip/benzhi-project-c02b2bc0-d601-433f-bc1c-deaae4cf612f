package application

import "blast-permit/internal/domain"

type CreateCaseCommand struct {
	SiteName       string `json:"siteName"`
	WorkZone       string `json:"workZone"`
	IdempotencyKey string `json:"idempotencyKey"`
}
type AddTargetsCommand struct {
	ExpectedVersion int64         `json:"expectedVersion"`
	IdempotencyKey  string        `json:"idempotencyKey"`
	Targets         []TargetInput `json:"targets"`
}
type TargetInput struct {
	TargetType          string  `json:"targetType"`
	Name                string  `json:"name"`
	DistanceMeters      float64 `json:"distanceMeters"`
	AllowedPpvMmPerSec  float64 `json:"allowedPpvMmPerSec"`
	BaselinePpvMmPerSec float64 `json:"baselinePpvMmPerSec"`
	MeasurementNote     string  `json:"measurementNote"`
}
type CompleteBaselineCommand struct {
	ExpectedVersion int64  `json:"expectedVersion"`
	IdempotencyKey  string `json:"idempotencyKey"`
}
type SubmitRevisionCommand struct {
	ExpectedVersion     int64                      `json:"expectedVersion"`
	IdempotencyKey      string                     `json:"idempotencyKey"`
	HolePattern         string                     `json:"holePattern"`
	MaxChargePerDelayKg float64                    `json:"maxChargePerDelayKg"`
	DelaySequenceMs     []int                      `json:"delaySequenceMs"`
	InitiationDirection string                     `json:"initiationDirection"`
	PropagationK        float64                    `json:"propagationK"`
	PropagationAlpha    float64                    `json:"propagationAlpha"`
	RemediationNote     string                     `json:"remediationNote,omitempty"`
	FindingResolutions  []domain.FindingResolution `json:"findingResolutions,omitempty"`
}
type ReviewCommand struct {
	ExpectedVersion    int64                           `json:"expectedVersion"`
	IdempotencyKey     string                          `json:"idempotencyKey"`
	SensorPoints       []domain.SensorPoint            `json:"sensorPoints"`
	SampleRateHz       int                             `json:"sampleRateHz"`
	TriggerPpvMmPerSec float64                         `json:"triggerPpvMmPerSec"`
	EvacuationRule     string                          `json:"evacuationRule"`
	RemainingRisk      string                          `json:"remainingRisk"`
	Decision           string                          `json:"decision"`
	Reasons            []ReviewReasonInput             `json:"reasons,omitempty"`
	ReasonResolutions  []domain.ReviewReasonResolution `json:"reasonResolutions,omitempty"`
}
type ReviewReasonInput struct {
	Category         string `json:"category"`
	Description      string `json:"description"`
	TargetID         string `json:"targetId,omitempty"`
	RelatedTargetID  string `json:"relatedTargetId,omitempty"`
	Parameter        string `json:"parameter,omitempty"`
	RelatedParameter string `json:"relatedParameter,omitempty"`
	RequiredChange   string `json:"requiredChange"`
}
type IssuePermitCommand struct {
	ExpectedVersion int64  `json:"expectedVersion"`
	IdempotencyKey  string `json:"idempotencyKey"`
	ValidHours      int    `json:"validHours"`
}

type CaseResponse struct {
	CaseID            string           `json:"caseId"`
	State             domain.CaseState `json:"state"`
	Version           int64            `json:"version"`
	CurrentRevisionID string           `json:"currentRevisionId,omitempty"`
	Replay            bool             `json:"replay,omitempty"`
}
type RevisionResponse struct {
	CaseResponse
	Revision           domain.DesignRevision      `json:"revision"`
	Assessment         domain.AssessmentSnapshot  `json:"assessment"`
	FindingTransitions []domain.FindingTransition `json:"findingTransitions,omitempty"`
	PendingFindings    []domain.Finding           `json:"pendingFindings"`
}
type AddTargetsResponse struct {
	CaseResponse
	Targets []domain.ProtectedTarget `json:"targets"`
}
type BaselineResponse struct {
	CaseResponse
	Precheck domain.BaselinePrecheck `json:"precheck"`
}
type ReviewResponse struct {
	CaseResponse
	Review                 domain.ReviewRound `json:"review"`
	PendingReviewReasonIDs []string           `json:"pendingReviewReasonIds"`
}
type PermitResponse struct {
	CaseResponse
	Permit domain.IgnitionPermit `json:"permit"`
}
type VerificationResponse struct {
	PermitNumber    string   `json:"permitNumber"`
	CaseID          string   `json:"caseId"`
	Valid           bool     `json:"valid"`
	EvidenceDigest  string   `json:"evidenceDigest"`
	Status          string   `json:"status"`
	FaultComponents []string `json:"faultComponents,omitempty"`
}

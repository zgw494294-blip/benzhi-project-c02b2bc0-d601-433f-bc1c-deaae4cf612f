package domain

import "time"

type BlastCase struct {
	CaseID            string    `json:"caseId"`
	SiteName          string    `json:"siteName"`
	WorkZone          string    `json:"workZone"`
	State             CaseState `json:"state"`
	CurrentRevisionID string    `json:"currentRevisionId,omitempty"`
	Version           int64     `json:"version"`
	CreatedAt         time.Time `json:"createdAt"`
	UpdatedAt         time.Time `json:"updatedAt"`
}

type BaselineIssue struct {
	Code     string `json:"code"`
	TargetID string `json:"targetId,omitempty"`
	Field    string `json:"field,omitempty"`
	Message  string `json:"message"`
}

type BaselineTargetMargin struct {
	TargetID             string  `json:"targetId"`
	RemainingPpvMmPerSec float64 `json:"remainingPpvMmPerSec"`
	RemainingRatio       float64 `json:"remainingRatio"`
	RiskLevel            string  `json:"riskLevel"`
	ControlOrder         int     `json:"controlOrder"`
}

type BaselinePrecheck struct {
	CaseID          string                 `json:"caseId"`
	Ready           bool                   `json:"ready"`
	TypeCounts      map[string]int         `json:"typeCounts"`
	MissingTypes    []string               `json:"missingTypes"`
	Issues          []BaselineIssue        `json:"issues"`
	TargetMargins   []BaselineTargetMargin `json:"targetMargins"`
	ControlTargetID string                 `json:"controlTargetId,omitempty"`
}

type BaselineConfirmation struct {
	ConfirmedAt     time.Time      `json:"confirmedAt"`
	TypeCounts      map[string]int `json:"typeCounts"`
	TargetCount     int            `json:"targetCount"`
	ControlTargetID string         `json:"controlTargetId"`
	RiskCounts      map[string]int `json:"riskCounts"`
}

type ProtectedTarget struct {
	TargetID            string  `json:"targetId"`
	CaseID              string  `json:"caseId"`
	TargetType          string  `json:"targetType"`
	Name                string  `json:"name"`
	DistanceMeters      float64 `json:"distanceMeters"`
	AllowedPpvMmPerSec  float64 `json:"allowedPpvMmPerSec"`
	BaselinePpvMmPerSec float64 `json:"baselinePpvMmPerSec"`
	MeasurementNote     string  `json:"measurementNote"`
}

type DesignRevision struct {
	RevisionID          string                        `json:"revisionId"`
	CaseID              string                        `json:"caseId"`
	RevisionNumber      int                           `json:"revisionNumber"`
	ParentRevisionID    string                        `json:"parentRevisionId,omitempty"`
	HolePattern         string                        `json:"holePattern"`
	MaxChargePerDelayKg float64                       `json:"maxChargePerDelayKg"`
	DelaySequenceMs     []int                         `json:"delaySequenceMs"`
	InitiationDirection string                        `json:"initiationDirection"`
	PropagationK        float64                       `json:"propagationK"`
	PropagationAlpha    float64                       `json:"propagationAlpha"`
	RemediationNote     string                        `json:"remediationNote,omitempty"`
	Diff                RevisionDiff                  `json:"diff"`
	ReviewComparison    []ReviewRequirementComparison `json:"reviewComparison,omitempty"`
	SubmittedAt         time.Time                     `json:"submittedAt"`
}

type ParameterChange struct {
	Field        string   `json:"field"`
	OldValue     any      `json:"oldValue,omitempty"`
	NewValue     any      `json:"newValue"`
	NumericDelta *float64 `json:"numericDelta,omitempty"`
}

type RevisionDiff struct {
	Baseline                bool              `json:"baseline"`
	Changes                 []ParameterChange `json:"changes"`
	AffectedTargetIDs       []string          `json:"affectedTargetIds"`
	RecalculateAllTargets   bool              `json:"recalculateAllTargets"`
	DelayAssessmentAffected bool              `json:"delayAssessmentAffected"`
	ReviewAttention         []string          `json:"reviewAttention"`
}

type PredictedTarget struct {
	TargetID             string  `json:"targetId"`
	PredictedPpvMmPerSec float64 `json:"predictedPpvMmPerSec"`
	MarginMmPerSec       float64 `json:"marginMmPerSec"`
	Pass                 bool    `json:"pass"`
}

type Finding struct {
	FindingID string `json:"findingId"`
	Code      string `json:"code"`
	TargetID  string `json:"targetId,omitempty"`
	Message   string `json:"message"`
	Blocking  bool   `json:"blocking"`
}

type AllowedChargeResult struct {
	TargetID           string   `json:"targetId"`
	Status             string   `json:"status"`
	MaxAllowedChargeKg *float64 `json:"maxAllowedChargeKg,omitempty"`
	Control            bool     `json:"control"`
}

type AssessmentTargetInput struct {
	TargetID           string  `json:"targetId"`
	DistanceMeters     float64 `json:"distanceMeters"`
	AllowedPpvMmPerSec float64 `json:"allowedPpvMmPerSec"`
}

type AssessmentInputSummary struct {
	RevisionID          string                  `json:"revisionId"`
	MaxChargePerDelayKg float64                 `json:"maxChargePerDelayKg"`
	PropagationK        float64                 `json:"propagationK"`
	PropagationAlpha    float64                 `json:"propagationAlpha"`
	FormulaVersion      string                  `json:"formulaVersion"`
	Targets             []AssessmentTargetInput `json:"targets"`
	AllowedCharges      []AllowedChargeResult   `json:"allowedCharges"`
	ControlTargetID     string                  `json:"controlTargetId,omitempty"`
}

type AssessmentSnapshot struct {
	AssessmentID        string                 `json:"assessmentId"`
	RevisionID          string                 `json:"revisionId"`
	InputDigest         string                 `json:"inputDigest"`
	InputSummary        AssessmentInputSummary `json:"inputSummary"`
	PredictedTargets    []PredictedTarget      `json:"predictedTargets"`
	Findings            []Finding              `json:"findings"`
	BlockingFindingIDs  []string               `json:"blockingFindingIds"`
	AllowedCharges      []AllowedChargeResult  `json:"allowedCharges"`
	ControlTargetID     string                 `json:"controlTargetId,omitempty"`
	ControlChargeKg     *float64               `json:"controlChargeKg,omitempty"`
	ChargeMarginKg      *float64               `json:"chargeMarginKg,omitempty"`
	ChargeMarginPercent *float64               `json:"chargeMarginPercent,omitempty"`
	FormulaVersion      string                 `json:"formulaVersion"`
	CalculatedAt        time.Time              `json:"calculatedAt"`
}

type FindingResolution struct {
	FindingID    string `json:"findingId"`
	HandlingNote string `json:"handlingNote"`
}

type FindingTransition struct {
	RevisionID    string    `json:"revisionId"`
	FromFindingID string    `json:"fromFindingId,omitempty"`
	ToFindingID   string    `json:"toFindingId,omitempty"`
	Status        string    `json:"status"`
	HandlingNote  string    `json:"handlingNote,omitempty"`
	CreatedAt     time.Time `json:"createdAt"`
}

type SensorPoint struct {
	Name     string `json:"name"`
	TargetID string `json:"targetId"`
	Location string `json:"location"`
}
type MonitoringPlan struct {
	PlanID             string               `json:"planId"`
	CaseID             string               `json:"caseId"`
	RevisionID         string               `json:"revisionId"`
	AssessmentID       string               `json:"assessmentId"`
	SensorPoints       []SensorPoint        `json:"sensorPoints"`
	SampleRateHz       int                  `json:"sampleRateHz"`
	TriggerPpvMmPerSec float64              `json:"triggerPpvMmPerSec"`
	EvacuationRule     string               `json:"evacuationRule"`
	RemainingRisk      string               `json:"remainingRisk"`
	ReviewDecision     string               `json:"reviewDecision"`
	ReviewedBy         string               `json:"reviewedBy"`
	ReviewedAt         time.Time            `json:"reviewedAt"`
	Validation         MonitoringValidation `json:"validation"`
}

type CoverageEntry struct {
	TargetID    string   `json:"targetId"`
	Required    bool     `json:"required"`
	Reasons     []string `json:"reasons"`
	SensorNames []string `json:"sensorNames"`
	Covered     bool     `json:"covered"`
}

type MonitoringIssue struct {
	Code     string  `json:"code"`
	TargetID string  `json:"targetId,omitempty"`
	Message  string  `json:"message"`
	Limit    float64 `json:"limit,omitempty"`
	Actual   float64 `json:"actual,omitempty"`
}

type MonitoringValidation struct {
	Ready              bool              `json:"ready"`
	CoverageMatrix     []CoverageEntry   `json:"coverageMatrix"`
	UncoveredTargetIDs []string          `json:"uncoveredTargetIds"`
	ThresholdIssues    []MonitoringIssue `json:"thresholdIssues"`
	Attention          []MonitoringIssue `json:"attention"`
}

type ReviewReason struct {
	ReasonID       string `json:"reasonId"`
	Category       string `json:"category"`
	Description    string `json:"description"`
	TargetID       string `json:"targetId,omitempty"`
	Parameter      string `json:"parameter,omitempty"`
	RequiredChange string `json:"requiredChange"`
}

type ReviewReasonResolution struct {
	ReasonID  string `json:"reasonId"`
	Confirmed bool   `json:"confirmed"`
	Note      string `json:"note,omitempty"`
}

type ReviewRequirementComparison struct {
	ReasonID       string   `json:"reasonId"`
	Responded      bool     `json:"responded"`
	MatchingFields []string `json:"matchingFields"`
}

type ReviewRound struct {
	Round             int                           `json:"round"`
	RevisionID        string                        `json:"revisionId"`
	AssessmentID      string                        `json:"assessmentId"`
	Plan              MonitoringPlan                `json:"plan"`
	Decision          string                        `json:"decision"`
	Reasons           []ReviewReason                `json:"reasons"`
	ReasonResolutions []ReviewReasonResolution      `json:"reasonResolutions"`
	Comparison        []ReviewRequirementComparison `json:"comparison"`
	Validation        MonitoringValidation          `json:"validation"`
}

type EvidenceComponent struct {
	Name   string `json:"name"`
	Digest string `json:"digest"`
	Status string `json:"status"`
}

type EvidenceIssue struct {
	Code      string `json:"code"`
	Component string `json:"component,omitempty"`
	Message   string `json:"message"`
}

type PermitPrecheck struct {
	CaseID          string              `json:"caseId"`
	Ready           bool                `json:"ready"`
	Components      []EvidenceComponent `json:"components"`
	EvidenceDigest  string              `json:"evidenceDigest,omitempty"`
	AuditSequence   int64               `json:"auditSequence"`
	AuditHeadDigest string              `json:"auditHeadDigest,omitempty"`
	Blockers        []EvidenceIssue     `json:"blockers"`
}

type AuditChainState struct {
	Continuous bool
	Sequence   int64
	HeadDigest string
	Digests    map[int64]string
	EventTypes map[int64]string
}

type IgnitionPermit struct {
	PermitNumber          string              `json:"permitNumber"`
	CaseID                string              `json:"caseId"`
	FrozenRevisionID      string              `json:"frozenRevisionId"`
	EvidenceDigest        string              `json:"evidenceDigest"`
	VerificationDigest    string              `json:"verificationDigest"`
	IssuedBy              string              `json:"issuedBy"`
	IssuedAt              time.Time           `json:"issuedAt"`
	ValidUntil            time.Time           `json:"validUntil"`
	VerificationStatus    string              `json:"verificationStatus"`
	FrozenComponents      []EvidenceComponent `json:"frozenComponents"`
	FrozenAuditSequence   int64               `json:"frozenAuditSequence"`
	FrozenAuditHeadDigest string              `json:"frozenAuditHeadDigest"`
}

type AuditEvent struct {
	Sequence  int64          `json:"sequence"`
	EventType string         `json:"eventType"`
	ActorRole string         `json:"actorRole"`
	ActorName string         `json:"actorName"`
	Details   map[string]any `json:"details"`
	CreatedAt time.Time      `json:"createdAt"`
	Digest    string         `json:"digest"`
}

type CaseFile struct {
	Case                   BlastCase             `json:"case"`
	Targets                []ProtectedTarget     `json:"targets"`
	Revisions              []DesignRevision      `json:"revisions"`
	Assessments            []AssessmentSnapshot  `json:"assessments"`
	BaselineConfirmation   *BaselineConfirmation `json:"baselineConfirmation,omitempty"`
	FindingHistory         []FindingTransition   `json:"findingHistory"`
	CurrentPendingFindings []Finding             `json:"currentPendingFindings"`
	Reviews                []ReviewRound         `json:"reviews"`
	CurrentReviewReasons   []ReviewReason        `json:"currentReviewReasons"`
	MonitoringPlan         *MonitoringPlan       `json:"monitoringPlan,omitempty"`
	Permit                 *IgnitionPermit       `json:"permit,omitempty"`
	AuditState             AuditChainState       `json:"-"`
}

func (f *CaseFile) CurrentRevision() *DesignRevision {
	for i := len(f.Revisions) - 1; i >= 0; i-- {
		if f.Revisions[i].RevisionID == f.Case.CurrentRevisionID {
			return &f.Revisions[i]
		}
	}
	return nil
}
func (f *CaseFile) CurrentAssessment() *AssessmentSnapshot {
	r := f.CurrentRevision()
	if r == nil {
		return nil
	}
	for i := len(f.Assessments) - 1; i >= 0; i-- {
		if f.Assessments[i].RevisionID == r.RevisionID {
			return &f.Assessments[i]
		}
	}
	return nil
}
func (f *CaseFile) CurrentReview() *ReviewRound {
	if len(f.Reviews) == 0 {
		return nil
	}
	return &f.Reviews[len(f.Reviews)-1]
}

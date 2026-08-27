package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"
)

func Digest(v any) string {
	b, _ := json.Marshal(v)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
func EvidenceDigest(f CaseFile) string {
	components := EvidenceComponents(f)
	return TotalEvidenceDigest(f.Case.CaseID, components)
}

func EvidenceComponents(f CaseFile) []EvidenceComponent {
	monitoringEvidence := struct {
		Plan   *MonitoringPlan
		Review *ReviewRound
	}{f.MonitoringPlan, f.CurrentReview()}
	monitoringPresent := f.MonitoringPlan != nil && f.CurrentReview() != nil
	values := []struct {
		name  string
		value any
		ok    bool
	}{
		{"baseline", struct {
			Targets      []ProtectedTarget
			Confirmation *BaselineConfirmation
		}{f.Targets, f.BaselineConfirmation}, f.BaselineConfirmation != nil && len(f.Targets) > 0},
		{"revision", f.CurrentRevision(), f.CurrentRevision() != nil},
		{"assessment", f.CurrentAssessment(), f.CurrentAssessment() != nil},
		{"monitoringPlan", monitoringEvidence, monitoringPresent},
	}
	components := make([]EvidenceComponent, 0, len(values))
	for _, value := range values {
		component := EvidenceComponent{Name: value.name, Status: "missing", Digest: Digest(value.value)}
		if value.ok {
			component.Status = "present"
		}
		components = append(components, component)
	}
	return components
}

func TotalEvidenceDigest(caseID string, components []EvidenceComponent) string {
	return Digest(struct {
		CaseID     string
		Components []EvidenceComponent
	}{caseID, components})
}
func PermitVerificationDigest(number, caseID, evidence string) string {
	return Digest(struct{ Number, CaseID, Evidence string }{number, caseID, evidence})
}

func PermitRecordDigest(number, caseID, evidence, revisionID string, issuedAt, validUntil time.Time, auditSequence int64, auditHead string) string {
	return Digest(struct {
		Number, CaseID, Evidence, RevisionID string
		IssuedAt, ValidUntil                 time.Time
		AuditSequence                        int64
		AuditHead                            string
	}{number, caseID, evidence, revisionID, issuedAt, validUntil, auditSequence, auditHead})
}

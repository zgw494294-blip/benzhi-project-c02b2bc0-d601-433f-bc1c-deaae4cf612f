package domain

func NewFinding(revisionID, code, targetID, message string) Finding {
	id := Digest(struct{ RevisionID, Code, TargetID string }{revisionID, code, targetID})[:20]
	return Finding{FindingID: "find_" + id, Code: code, TargetID: targetID, Message: message, Blocking: true}
}
func HasBlockers(a AssessmentSnapshot) bool { return len(a.BlockingFindingIDs) > 0 }

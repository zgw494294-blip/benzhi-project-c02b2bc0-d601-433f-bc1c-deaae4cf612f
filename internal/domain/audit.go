package domain

func AuditDigest(previous string, sequence int64, eventType, actorRole, actorName string, details map[string]any) string {
	return Digest(struct {
		Previous                        string
		Sequence                        int64
		EventType, ActorRole, ActorName string
		Details                         map[string]any
	}{previous, sequence, eventType, actorRole, actorName, details})
}

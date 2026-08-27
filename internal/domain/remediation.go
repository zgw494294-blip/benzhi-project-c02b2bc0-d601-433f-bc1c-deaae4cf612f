package domain

import "time"

func MergeFindingLifecycle(previous, current AssessmentSnapshot, resolutions []FindingResolution, revisionID string, at time.Time) []FindingTransition {
	notes := map[string]string{}
	for _, resolution := range resolutions {
		notes[resolution.FindingID] = resolution.HandlingNote
	}
	currentByKey := map[string]Finding{}
	previousKeys := map[string]bool{}
	for _, finding := range current.Findings {
		if finding.Blocking {
			currentByKey[finding.Code+"\x00"+finding.TargetID] = finding
		}
	}
	for _, finding := range previous.Findings {
		if finding.Blocking {
			previousKeys[finding.Code+"\x00"+finding.TargetID] = true
		}
	}
	transitions := []FindingTransition{}
	for _, finding := range previous.Findings {
		if !finding.Blocking {
			continue
		}
		key := finding.Code + "\x00" + finding.TargetID
		transition := FindingTransition{RevisionID: revisionID, FromFindingID: finding.FindingID, HandlingNote: notes[finding.FindingID], CreatedAt: at}
		if successor, ok := currentByKey[key]; ok {
			transition.Status = "still_open"
			transition.ToFindingID = successor.FindingID
		} else {
			transition.Status = "closed"
			for _, candidate := range current.Findings {
				if candidate.Blocking && !previousKeys[candidate.Code+"\x00"+candidate.TargetID] && (candidate.TargetID == finding.TargetID || candidate.TargetID == "" || finding.TargetID == "") {
					transition.Status = "replaced"
					transition.ToFindingID = candidate.FindingID
					break
				}
			}
		}
		transitions = append(transitions, transition)
	}
	for _, finding := range current.Findings {
		if finding.Blocking && !previousKeys[finding.Code+"\x00"+finding.TargetID] {
			transitions = append(transitions, FindingTransition{RevisionID: revisionID, ToFindingID: finding.FindingID, Status: "new", CreatedAt: at})
		}
	}
	return transitions
}

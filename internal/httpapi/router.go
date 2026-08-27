package httpapi

import "net/http"

func (a *API) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", a.Health)
	mux.HandleFunc("POST /api/v1/cases", a.CreateCase)
	mux.HandleFunc("GET /api/v1/cases/{caseId}", a.GetCase)
	mux.HandleFunc("POST /api/v1/cases/{caseId}/targets", a.AddTargets)
	mux.HandleFunc("GET /api/v1/cases/{caseId}/baseline", a.BaselinePrecheck)
	mux.HandleFunc("GET /api/v1/cases/{caseId}/baseline/precheck", a.BaselinePrecheck)
	mux.HandleFunc("POST /api/v1/cases/{caseId}/baseline/complete", a.CompleteBaseline)
	mux.HandleFunc("POST /api/v1/cases/{caseId}/revisions", a.SubmitRevision)
	mux.HandleFunc("POST /api/v1/cases/{caseId}/remediations", a.SubmitRemediation)
	mux.HandleFunc("POST /api/v1/cases/{caseId}/reviews", a.ReviewCase)
	mux.HandleFunc("POST /api/v1/cases/{caseId}/permit", a.IssuePermit)
	mux.HandleFunc("GET /api/v1/cases/{caseId}/permit", a.PermitPrecheck)
	mux.HandleFunc("GET /api/v1/cases/{caseId}/permit/precheck", a.PermitPrecheck)
	mux.HandleFunc("GET /api/v1/cases/{caseId}/audit", a.GetAudit)
	mux.HandleFunc("GET /api/v1/permits/{permitNumber}/verify", a.VerifyPermit)
	return recoverMiddleware(jsonMiddleware(mux))
}

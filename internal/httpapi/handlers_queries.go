package httpapi

import "net/http"

func (a *API) GetCase(w http.ResponseWriter, r *http.Request) {
	out, err := a.service.GetCase(r.Context(), r.PathValue("caseId"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}
func (a *API) GetAudit(w http.ResponseWriter, r *http.Request) {
	out, err := a.service.Audit(r.Context(), r.PathValue("caseId"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"caseId": r.PathValue("caseId"), "events": out})
}
func (a *API) BaselinePrecheck(w http.ResponseWriter, r *http.Request) {
	out, err := a.service.BaselinePrecheck(r.Context(), r.PathValue("caseId"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}
func (a *API) PermitPrecheck(w http.ResponseWriter, r *http.Request) {
	out, err := a.service.PermitPrecheck(r.Context(), r.PathValue("caseId"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}
func (a *API) VerifyPermit(w http.ResponseWriter, r *http.Request) {
	out, err := a.service.VerifyPermit(r.Context(), r.PathValue("permitNumber"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

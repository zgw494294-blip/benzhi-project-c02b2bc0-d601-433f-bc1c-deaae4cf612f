package httpapi

import (
	"net/http"

	"blast-permit/internal/domain"
)

func (a *API) GetCase(w http.ResponseWriter, r *http.Request) {
	caseID := r.PathValue("caseId")
	if cached, ok := a.cachedCase(caseID); ok {
		writeJSON(w, http.StatusOK, cached)
		return
	}
	out, err := a.service.GetCase(r.Context(), caseID)
	if err != nil {
		writeError(w, err)
		return
	}
	a.rememberCase(caseID, out)
	writeJSON(w, http.StatusOK, out)
}

func (a *API) cachedCase(caseID string) (*domain.CaseFile, bool) {
	a.caseMu.RLock()
	defer a.caseMu.RUnlock()
	file, ok := a.caseCache[caseID]
	return file, ok
}

func (a *API) rememberCase(caseID string, file *domain.CaseFile) {
	a.caseMu.Lock()
	defer a.caseMu.Unlock()
	a.caseCache[caseID] = file
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

package handlers

import (
	"encoding/json"
	"net/http"
	"project-management-app/microservices/projects-service/domain"
	"strings"
)

func writeErrorResp(err error, w http.ResponseWriter) {
	if err == nil {
		return
	} else if err.Error() == domain.ErrUnauthorized().Error() {
		w.WriteHeader(http.StatusForbidden)
	} else if strings.Contains(err.Error(), "not found") {
		w.WriteHeader(http.StatusNotFound)
	} else {
		w.WriteHeader(http.StatusInternalServerError)
	}
	w.Write([]byte(err.Error()))
}

func writeResp(resp any, _ int, w http.ResponseWriter) {
	w.WriteHeader(http.StatusCreated)
	if resp == nil {
		return
	}
	respBytes, err := json.Marshal(resp)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Add("Content-Type", "application/json")
	w.Write(respBytes)
}

func readReq(req any, r *http.Request, w http.ResponseWriter) error {
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
	}
	return err
}

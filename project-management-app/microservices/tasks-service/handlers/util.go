package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"project-management-app/microservices/projects-service/domain"
	"strings"
)

func writeErrorResp(err error, w http.ResponseWriter) {
	if err == nil {
		return
	}

	switch {
	case err.Error() == domain.ErrUnauthorized().Error():
		w.WriteHeader(http.StatusForbidden)
	case strings.Contains(err.Error(), "not found"):
		w.WriteHeader(http.StatusNotFound)
	case strings.Contains(err.Error(), "cannot remove member from a finished task"):
		w.WriteHeader(http.StatusBadRequest)
	default:
		log.Printf("Unexpected error: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")

	if _, writeErr := w.Write([]byte(err.Error())); writeErr != nil {
		log.Printf("Error writing response: %v", writeErr)
	}
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

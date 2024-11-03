package handlers

import (
	"context"
	"net/http"
	"project-management-app/microservices/projects-service/domain"
	"project-management-app/microservices/projects-service/services"
	"strconv"
)

type KeyProduct struct{}

type ProjectHandler struct {
	projects services.ProjectService
}

func NewprojectHandler(projects services.ProjectService) (ProjectHandler, error) {
	return ProjectHandler{
		projects: projects,
	}, nil
}

func (h ProjectHandler) Create(w http.ResponseWriter, r *http.Request) {

	req := &struct {
		ManagerId  string
		Name       string
		EndDate    string
		MinWorkers int
		MaxWorkers int
	}{}
	err := readReq(req, r, w)
	if err != nil {
		return
	}

	project, err := h.projects.Create(req.ManagerId, req.Name, req.EndDate, req.MinWorkers, req.MaxWorkers)
	if err != nil {
		writeErrorResp(err, w)
		return
	}

	resp := struct {
		Id              string
		ManagerId       string
		ManagerUsername string
		Name            string
		EndDate         string
		MinWorkers      string
		MaxWorkers      string
	}{
		Id:              project.Id.String(),
		ManagerId:       project.Manager.Id.String(),
		ManagerUsername: project.Manager.Username,
		Name:            project.Name,
		EndDate:         project.EndDate.String(),
		MinWorkers:      strconv.Itoa(project.MinWorkers),
		MaxWorkers:      strconv.Itoa(project.MaxWorkers),
	}
	writeResp(resp, http.StatusCreated, w)

}

func (u *ProjectHandler) MiddlewareContentTypeSet(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, h *http.Request) {

		rw.Header().Add("Content-Type", "application/json")

		next.ServeHTTP(rw, h)
	})
}

func (u *ProjectHandler) ProjectContextMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, h *http.Request) {

		project := &domain.Project{}
		err := project.FromJSON(h.Body)

		if err != nil {
			http.Error(rw, "Unable to decode json", http.StatusBadRequest)
			return
		}

		ctx := context.WithValue(h.Context(), KeyProduct{}, project)
		h = h.WithContext(ctx)
		next.ServeHTTP(rw, h)
	})
}

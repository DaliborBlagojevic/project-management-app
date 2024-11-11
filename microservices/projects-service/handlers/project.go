package handlers

import (
	"context"
	"net/http"
	"project-management-app/microservices/projects-service/domain"
	"project-management-app/microservices/projects-service/services"
	"strconv"

	"github.com/gorilla/mux"
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
		ManagerUsername string
		Name            string
		EndDate         string
		MinWorkers      int
		MaxWorkers      int
	}{}
	err := readReq(req, r, w)
	if err != nil {
		return
	}

	project, err := h.projects.Create(req.ManagerUsername, req.Name, req.EndDate, req.MinWorkers, req.MaxWorkers)
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

func (h ProjectHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	// Dohvatamo sve projekte koristeći ProjectService
	projects, err := h.projects.GetAll()
	if err != nil {
		writeErrorResp(err, w)
		return
	}

	// Pripremamo odgovor kao slice mapiranih struktura
	resp := make([]struct {
		Id              string `json:"id"`
		ManagerId       string `json:"managerId"`
		ManagerUsername string `json:"managerUsername"`
		Name            string `json:"name"`
		EndDate         string `json:"endDate"`
		MinWorkers      int    `json:"minWorkers"`
		MaxWorkers      int    `json:"maxWorkers"`
	}, len(projects))

	for i, project := range projects {
		resp[i] = struct {
			Id              string `json:"id"`
			ManagerId       string `json:"managerId"`
			ManagerUsername string `json:"managerUsername"`
			Name            string `json:"name"`
			EndDate         string `json:"endDate"`
			MinWorkers      int    `json:"minWorkers"`
			MaxWorkers      int    `json:"maxWorkers"`
		}{
			Id:              project.Id.Hex(),
			ManagerId:       project.Manager.Id.Hex(),
			ManagerUsername: project.Manager.Username,
			Name:            project.Name,
			EndDate:         project.EndDate.Format("2006-01-02"),
			MinWorkers:      project.MinWorkers,
			MaxWorkers:      project.MaxWorkers,
		}
	}

	// Šaljemo odgovor kao JSON sa statusom 200 OK
	writeResp(resp, http.StatusOK, w)
}

func (h ProjectHandler) AddMember(w http.ResponseWriter, r *http.Request) {
	// Dohvatamo ID projekta iz URL parametra
	vars := mux.Vars(r)
	projectId := vars["id"]

	// Dohvatamo korisnika iz JSON tela zahteva
	user := &domain.User{}
	err := user.FromJSON(r.Body)
	if err != nil {
		writeErrorResp(err, w)
		return
	}

	// Dodajemo korisnika u projekat koristeći ProjectService
	err = h.projects.AddMember(projectId, *user)
	if err != nil {
		writeErrorResp(err, w)
		return
	}

	// Šaljemo odgovor sa statusom 200 OK
	writeResp(nil, http.StatusOK, w)
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

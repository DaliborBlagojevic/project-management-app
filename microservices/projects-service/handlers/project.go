package handlers

import (
	"context"
	"log"
	"net/http"
	"project-management-app/microservices/projects-service/domain"
	"project-management-app/microservices/projects-service/repositories"
	"project-management-app/microservices/projects-service/services"

	"github.com/gorilla/mux"
)

type KeyProduct struct{}

type ProjectHandler struct {
	projects *services.ProjectService
	repo     *repositories.ProjectRepo
}

func NewprojectHandler(s *services.ProjectService, r *repositories.ProjectRepo) *ProjectHandler {
	return &ProjectHandler{s, r}
}

func (p *ProjectHandler) Create(rw http.ResponseWriter, h *http.Request) {
	project := h.Context().Value(KeyProduct{}).(*domain.Project)
	p.repo.Create(project)
	rw.WriteHeader(http.StatusCreated)
}

func (p *ProjectHandler) GetAll(rw http.ResponseWriter, h *http.Request) {
	projects, err := p.repo.GetAll()
	if err != nil {
		log.Print("Database exception: ", err)
	}

	if projects == nil {
		return
	}

	err = projects.ToJSON(rw)
	if err != nil {
		http.Error(rw, "Unable to convert to json", http.StatusInternalServerError)
		log.Fatal("Unable to convert to json :", err)
		return
	}
}

func (p *ProjectHandler) GetByID(rw http.ResponseWriter, h *http.Request) {
	vars := mux.Vars(h)
	id := vars["id"]

	project, err := p.repo.GetById(id)

	if err != nil {
		log.Print("Database exception: ", err)
	}

	if project == nil {
		return
	}

	err = project.ToJSON(rw)
	if err != nil {
		http.Error(rw, "Unable to convert to json", http.StatusInternalServerError)
		log.Fatal("Unable to convert to json :", err)
		return
	}
}

func (h ProjectHandler) AddMember(w http.ResponseWriter, r *http.Request) {
	// Dohvatamo ID projekta iz URL parametra
	vars := mux.Vars(r)
	id := vars["id"]

	// Iz tela zahteva čitamo korisnika koji se dodaje i kreiramo domain.User objekat
	user := &domain.User{}
	err := user.FromJSON(r.Body)
	if err != nil {
		writeErrorResp(err, w)
		return
	}

	// Pozivamo ProjectService da doda korisnika u projekat
	err = h.projects.AddMember(id, *user)
	if err != nil {
		writeErrorResp(err, w)
		return
	}

	// Ako je sve prošlo OK, šaljemo prazan odgovor sa statusom 204 No Content
	w.WriteHeader(http.StatusNoContent)
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

func (p *ProjectHandler) MiddlewareUsersDeserialization(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, h *http.Request) {
		users := &domain.Users{}
		err := users.FromJSON(h.Body)
		if err != nil {
			http.Error(rw, "Unable to decode json", http.StatusBadRequest)
			log.Fatal(err)
			return
		}

		ctx := context.WithValue(h.Context(), KeyProduct{}, users)
		h = h.WithContext(ctx)

		next.ServeHTTP(rw, h)
	})
}

func (p *ProjectHandler) MiddlewareUserDeserialization(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, h *http.Request) {
		user := &domain.User{}
		err := user.FromJSON(h.Body)
		if err != nil {
			http.Error(rw, "Unable to decode json", http.StatusBadRequest)
			log.Fatal(err)
			return
		}

		ctx := context.WithValue(h.Context(), KeyProduct{}, user)
		h = h.WithContext(ctx)

		next.ServeHTTP(rw, h)
	})
}

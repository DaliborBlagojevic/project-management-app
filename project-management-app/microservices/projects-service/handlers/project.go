package handlers

import (
	"context"
	"log"
	"net/http"
	"project-management-app/microservices/projects-service/domain"
	"project-management-app/microservices/projects-service/repositories"
	"project-management-app/microservices/projects-service/services"

	authorizationlib "github.com/Bijelic03/authorizationlibGo"
	"github.com/gorilla/mux"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

type KeyProduct struct{}

type ProjectHandler struct {
	projects *services.ProjectService
	repo     *repositories.ProjectRepo
	tracer trace.Tracer
}

func NewprojectHandler(s *services.ProjectService, r *repositories.ProjectRepo, t trace.Tracer) *ProjectHandler {
	return &ProjectHandler{s, r, t}
}

func (p *ProjectHandler) Create(rw http.ResponseWriter, h *http.Request) {
	ctx, span := p.tracer.Start(h.Context(), "OrderHandler.GetOrder")
	defer span.End()
	project := h.Context().Value(KeyProduct{}).(*domain.Project)
	p.repo.Create(ctx, project)
	err := project.ToJSON(rw)
	if err != nil {
		http.Error(rw, "Unable to convert to json", http.StatusInternalServerError)
		log.Fatal("Unable to convert to json :", err)
		return
	}

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

func (p *ProjectHandler) GetProjectsByUser(rw http.ResponseWriter, h *http.Request) {
	ctx, span := p.tracer.Start(h.Context(), "OrderHandler.GetOrder")
	defer span.End()
	username := h.Context().Value(authorizationlib.UsernameKey).(string)
	role := h.Context().Value(authorizationlib.RoleKey).(string)

	var projects domain.Projects
	var err error

	switch role {
	case "PROJECT_MANAGER":
		projects, err = p.projects.GetProjectsByManager(ctx, username)
	case "PROJECT_MEMBER":
		projects, err = p.projects.GetProjectsByMember(ctx, username)
	default:
		http.Error(rw, "Invalid role provided", http.StatusBadRequest)
		return
	}

	if err != nil {
		span.SetStatus(codes.Error, err.Error())

		writeErrorResp(err, rw)
		return
	}

	if projects == nil {
		projects = domain.Projects{}
	}

	response := map[string]interface{}{
		"projects": projects,
		"role":     role,
	}

	writeResp(response, http.StatusOK, rw)
}

func (p *ProjectHandler) GetByID(rw http.ResponseWriter, h *http.Request) {
	vars := mux.Vars(h)
	id := vars["id"]

	username := h.Context().Value(authorizationlib.UsernameKey).(string)
	role := h.Context().Value(authorizationlib.RoleKey).(string)

	project, err := p.repo.GetById(id, username, role)
	if err != nil {
		log.Print("Database exception: ", err)
		http.Error(rw, err.Error(), http.StatusNotFound)
		return
	}

	response := map[string]interface{}{
		"project": project,
		"role":    role,
	}

	writeResp(response, http.StatusOK, rw)
}

func (p *ProjectHandler) GetMembersByID(rw http.ResponseWriter, h *http.Request) {
	ctx, span := p.tracer.Start(h.Context(), "ProjectsHandler.GetProjectsByManagerAndIsActive")
	defer span.End()
	vars := mux.Vars(h)
	id := vars["id"]

	project, err := p.repo.GetMembersByProjectId(ctx, id)

	if err != nil {
		http.Error(rw, err.Error(), http.StatusNoContent)
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

func (h ProjectHandler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	username := vars["username"]

	h.projects.RemoveMember(id, username)

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

func (p *ProjectHandler) GetProjectsByManagerAndIsActive(rw http.ResponseWriter, h *http.Request) {
	ctx, span := p.tracer.Start(h.Context(), "ProjectsHandler.GetProjectsByManagerAndIsActive")
	defer span.End()
	log.Println("zdravoo")
	vars := mux.Vars(h)
	username := vars["username"]

	projects, err := p.projects.GetProjectsByManagerAndIsActive(ctx, username)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		log.Printf("Error fetching projects for manager %s: %v", username, err)
		http.Error(rw, "Unable to fetch projects", http.StatusInternalServerError)
		return
	}

	if projects == nil {
		log.Printf("No projects found for manager %s", username)
		rw.WriteHeader(http.StatusNoContent)
		return
	}

	err = projects.ToJSON(rw)
	if err != nil {
		log.Printf("Error converting projects to JSON for manager %s: %v", username, err)
		http.Error(rw, "Unable to convert to json", http.StatusInternalServerError)
		return
	}
}

func (p *ProjectHandler) ExtractTraceInfoMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))
        traceID := trace.SpanContextFromContext(ctx).TraceID().String()
        log.Printf("Extracted TraceID: %s", traceID)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}


package handlers

import (
	"context"
	"errors"

	"log"
	"net/http"
	"project-management-app/microservices/projects-service/domain"
	"project-management-app/microservices/projects-service/repositories"
	"project-management-app/microservices/projects-service/services"
	"strconv"

	authorizationlib "github.com/Bijelic03/authorizationlibGo"
	"github.com/gorilla/mux"
	"go.opentelemetry.io/otel/trace"
)

type KeyTask struct{}

type TaskHandler struct {
	tasks *services.TaskService
	repo  *repositories.TaskRepo
	tracer trace.Tracer
}

func NewTaskHandler(s *services.TaskService, r *repositories.TaskRepo, t trace.Tracer) *TaskHandler {
	return &TaskHandler{s, r, t}
}

// Create - Kreira novi zadatak
func (h TaskHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx, span := h.tracer.Start(r.Context(), "TasksHandler.Create")
	defer span.End()

	req := &struct {
		Status      domain.Status `json:"status"`
		Name        string        `json:"name"`
		Description string        `json:"description"`
		ProjectId   string        `json:"project"`
	}{}

	err := readReq(req, r, w)
	if err != nil {
		return
	}

	task, err := h.tasks.Create(ctx, req.Status, req.Name, req.Description, req.ProjectId)
	if err != nil {
		writeErrorResp(err, w)
		return
	}

	resp := struct {
		Id          string `json:"id"`
		ProjectId   string `json:"project"`
		Name        string `json:"name"`
		Description string `json:"description"`
		Status      string `json:"status"`
	}{
		Id:          task.Id.Hex(),
		ProjectId:   task.Project,
		Name:        task.Name,
		Description: task.Description,
		Status:      strconv.Itoa(int(task.Status)),
	}
	writeResp(resp, http.StatusCreated, w)
}

func (h TaskHandler) Update(w http.ResponseWriter, r *http.Request) {
	req := &struct {
		Id          string        `json:"id"`
		Status      domain.Status `json:"status"`
		Name        string        `json:"name"`
		Description string        `json:"description"`
		ProjectId   string        `json:"project"`
	}{}

	err := readReq(req, r, w)
	if err != nil {
		log.Println(err.Error())
		return
	}

	username := r.Context().Value(authorizationlib.UsernameKey).(string)
	role := r.Context().Value(authorizationlib.RoleKey).(string)

	if role == "PROJECT_MEMBER" {
		task, err := h.repo.FindById(req.Id)
		if err != nil {
			writeErrorResp(err, w)
			return
		}

		isMember := false
		for _, member := range task.Members {
			if member.Username == username {
				isMember = true
				break
			}
		}

		if !isMember {
			writeErrorResp(errors.New("user is not authorized to update this task"), w)
			return
		}
	}

	task, err := h.tasks.Update(req.Id, req.Status, req.Name, req.Description, req.ProjectId)
	if err != nil {
		writeErrorResp(err, w)
		return
	}

	resp := struct {
		Id          string `json:"id"`
		ProjectId   string `json:"project"`
		Name        string `json:"name"`
		Description string `json:"description"`
		Status      string `json:"status"`
	}{
		Id:          task.Id.Hex(),
		ProjectId:   task.Project,
		Name:        task.Name,
		Description: task.Description,
		Status:      strconv.Itoa(int(task.Status)),
	}

	writeResp(resp, http.StatusCreated, w)
}

func (h *TaskHandler) FilterMembersNotOnTask(w http.ResponseWriter, r *http.Request) {
	// Dohvati project ID iz URL parametra
	vars := mux.Vars(r)
	projectId := vars["projectId"]
	taskId := vars["taskId"]

	// Pozivamo servis da filtrira članove koji nisu već na zadatku
	members, err := h.tasks.FilterMembersNotOnTask(projectId, taskId)
	if err != nil {
		writeErrorResp(err, w)
		return
	}

	// Vraćamo filtrirane članove kao JSON
	err = members.ToJSON(w)
	if err != nil {
		http.Error(w, "Unable to convert to json", http.StatusInternalServerError)
		log.Fatal("Unable to convert to json :", err)
		return
	}
}

func (p *TaskHandler) GetAll(rw http.ResponseWriter, h *http.Request) {
	tasks, err := p.repo.GetAll()
	if err != nil {
		log.Print("Database exception: ", err)
	}

	if tasks == nil {
		return
	}

	err = tasks.ToJSON(rw)
	if err != nil {
		http.Error(rw, "Unable to convert to json", http.StatusInternalServerError)
		log.Fatal("Unable to convert to json :", err)
		return
	}
}

func (p *TaskHandler) GetTasksByProject(rw http.ResponseWriter, h *http.Request) {
	ctx, span := p.tracer.Start(h.Context(), "TasksHandler.GetTasksByProject")
	defer span.End()

	vars := mux.Vars(h)
	id := vars["id"]

	tasks, err := p.repo.GetByProject( ctx ,id)
	if err != nil {
		log.Print("Database exception: ", err)
	}

	if tasks == nil {
		return
	}

	err = tasks.ToJSON(rw)
	if err != nil {
		http.Error(rw, "Unable to convert to json", http.StatusInternalServerError)
		log.Fatal("Unable to convert to json :", err)
		return
	}
}

func (p *TaskHandler) GetMembersByID(rw http.ResponseWriter, h *http.Request) {
	vars := mux.Vars(h)
	id := vars["id"]

	project, err := p.repo.GetMembersByTaskId(id)

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

func (h TaskHandler) AddMember(w http.ResponseWriter, r *http.Request) {
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
	err = h.tasks.AddMember(id, *user)
	if err != nil {
		writeErrorResp(err, w)
		return
	}

	// Ako je sve prošlo OK, šaljemo prazan odgovor sa statusom 204 No Content
	w.WriteHeader(http.StatusNoContent)
}

func (h TaskHandler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskId := vars["taskId"]

	user := &domain.User{}
	err := user.FromJSON(r.Body)
	if err != nil {
		writeErrorResp(err, w)
		return
	}

	// Validacija: Proveri da li task ima status završen
	task, err := h.repo.FindById(taskId)
	if err != nil {
		writeErrorResp(err, w)
		return
	}
	if task.Status == domain.FINISHED {
		err := errors.New("cannot remove member from a finished task")
		writeErrorResp(err, w)
		return
	}

	err = h.tasks.RemoveMember(taskId, *user)
	if err != nil {
		writeErrorResp(err, w)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (u *TaskHandler) MiddlewareContentTypeSet(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, h *http.Request) {

		rw.Header().Add("Content-Type", "application/json")

		next.ServeHTTP(rw, h)
	})
}

func (u *TaskHandler) ProjectContextMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, h *http.Request) {

		project := &domain.Task{}
		err := project.FromJSON(h.Body)

		if err != nil {
			http.Error(rw, "Unable to decode json", http.StatusBadRequest)
			return
		}

		ctx := context.WithValue(h.Context(), KeyTask{}, project)
		h = h.WithContext(ctx)
		next.ServeHTTP(rw, h)
	})
}

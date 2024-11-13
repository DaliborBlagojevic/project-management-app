package handlers

import (
	"context"
	"net/http"
	"project-management-app/microservices/projects-service/domain"
	"project-management-app/microservices/projects-service/services"
	"strconv"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type KeyTask struct{}

type TaskHandler struct {
	tasks services.TaskService
}

func NewTaskHandler(tasks services.TaskService) (TaskHandler, error) {
	return TaskHandler{
		tasks: tasks,
	}, nil
}

// Create - Kreira novi zadatak
func (h TaskHandler) Create(w http.ResponseWriter, r *http.Request) {


	req := &struct {
		Status      domain.Status `json:"status"`
		Name        string        `json:"name"`
		Description string        `json:"description"`
		ProjectId   string        `json:"projectId"`
	}{}

	err := readReq(req, r, w)
	if err != nil {
		return
	}

	projectID, err := primitive.ObjectIDFromHex(req.ProjectId)
	if err != nil {
		writeErrorResp(err, w)
		return
	}

	task, err := h.tasks.Create(req.Status, req.Name, req.Description, projectID)
	if err != nil {
		writeErrorResp(err, w)
		return
	}

	resp := struct {
		Id          string `json:"id"`
		ProjectId   string `json:"projectId"`
		Name        string `json:"name"`
		Description string `json:"description"`
		Status      string `json:"status"`
	}{
		Id:          task.Id.Hex(),
		ProjectId:   task.Project.Hex(),
		Name:        task.Name,
		Description: task.Description,
		Status:      strconv.Itoa(int(task.Status)),
	}
	writeResp(resp, http.StatusCreated, w)
}




func (u *TaskHandler) MiddlewareContentTypeSet(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, h *http.Request) {

		rw.Header().Add("Content-Type", "application/json")

		next.ServeHTTP(rw, h)
	})
}

func (u *TaskHandler) ProjectContextMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, h *http.Request) {

		project := &domain.Project{}
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





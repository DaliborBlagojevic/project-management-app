package services

import (
	"encoding/json"
	"fmt"
	"net/http"
	"project-management-app/microservices/projects-service/domain"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type TaskService struct {
	tasks domain.TaskRepository
}

func NewTaskService(tasks domain.TaskRepository) (TaskService, error) {
	return TaskService{
		tasks: tasks,
	}, nil
}




// Create - Kreira novi zadatak sa prosleđenim parametrima
func (s TaskService) Create(status domain.Status, name string, description string, projectID primitive.ObjectID) (domain.Task, error) {

	task := domain.Task{
		Id:          primitive.NewObjectID(),
		Project:     projectID,
		Name:        name,
		Description: description,
		Status:      status,
	}

	return s.tasks.Insert(task)
}

// GetUser - Dohvata korisnika po korisničkom imenu
func (s *TaskService) GetUser(username string) (domain.User, error) {

	url := fmt.Sprintf("http://user-server:8080/users/%s", username)

	resp, err := http.Get(url)
	if err != nil {
		return domain.User{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return domain.User{}, fmt.Errorf("user not found")
	}

	var user domain.User
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return domain.User{}, fmt.Errorf("failed to decode user: %v", err)
	}

	return user, nil
}

package services

import (
	"errors"
	"project-management-app/microservices/projects-service/domain"
	"project-management-app/microservices/projects-service/repositories"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type TaskService struct {
	tasks *repositories.TaskRepo
}

func NewTaskService(tasks *repositories.TaskRepo) *TaskService {
	return &TaskService{tasks}
}




// Create - Kreira novi zadatak sa prosleđenim parametrima
func (s TaskService) Create(status domain.Status, name string, description string, projectID primitive.ObjectID) (domain.Task, error) {
	// Proveri da li već postoji zadatak sa istim imenom
	existingTask, err := s.tasks.FindByName(name)
	if err != nil {
		return domain.Task{}, err
	}
	if existingTask != nil {
		return domain.Task{}, errors.New("zadatak sa istim imenom već postoji")
	}

	// Kreiraj novi zadatak
	task := domain.Task{
		Id:          primitive.NewObjectID(),
		Project:     projectID,
		Name:        name,
		Description: description,
		Status:      1,
	}

	return s.tasks.Insert(task)
}



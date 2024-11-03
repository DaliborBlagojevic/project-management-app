package services

import (
	"project-management-app/microservices/projects-service/domain"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ProjectService struct {
	projects domain.ProjectRepository
}

func NewProjectService(projects domain.ProjectRepository) (ProjectService, error) {
	return ProjectService{
		projects: projects,
	}, nil
}

func (s ProjectService) Create(managerId string, name string, endDate string, minWorkers int, maxWorkers int) (domain.Project, error) {
	managerIDHex := "64b4d5b2d4a1b3c2d5f7e8a9"

	managerID, err := primitive.ObjectIDFromHex(managerIDHex)
	if err != nil {
		return domain.Project{}, err
	}

	manager := domain.User{
		Id: managerID,
	}

	parsedEndDate, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		return domain.Project{}, err
	}

	project := domain.Project{
		Id:         primitive.NewObjectID(),
		Manager:    manager,
		Members:    []domain.User{},
		Name:       name,
		EndDate:    parsedEndDate,
		MinWorkers: minWorkers,
		MaxWorkers: maxWorkers,
	}

	return s.projects.Insert(project)
}

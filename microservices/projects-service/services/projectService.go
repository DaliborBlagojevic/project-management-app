package services

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
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

func (s ProjectService) Create(managerUsername string, name string, endDate string, minWorkers int, maxWorkers int) (domain.Project, error) {

	var manager domain.User

	manager, err := s.GetUser(managerUsername)
	if err != nil {
		return domain.Project{}, err
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

func (s *ProjectService) GetUser(username string) (domain.User, error) {

	url := fmt.Sprintf("http://user-server:8080/users/%s", username)

	resp, err := http.Get(url)
	if err != nil {
		return domain.User{}, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return domain.User{}, err
	}

	var users []domain.User
	if err := json.Unmarshal(body, &users); err != nil {
		return domain.User{}, fmt.Errorf("failed to decode user list: %v", err)
	}

	if len(users) == 0 {
		return domain.User{}, fmt.Errorf("no user found")
	}

	return users[0], nil
}

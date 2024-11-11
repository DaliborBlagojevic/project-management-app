package services

import (
	"encoding/json"
	"fmt"
	"io"
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
		Name:       name,
		EndDate:    parsedEndDate,
		MinWorkers: minWorkers,
		MaxWorkers: maxWorkers,
		Members:    domain.Users{},
	}

	return s.projects.Insert(project)
}

func (s ProjectService) GetAll() (domain.Projects, error) {
	projects, err := s.projects.GetAll()
	if err != nil {
		return nil, fmt.Errorf("error fetching projects: %v", err)
	}
	return projects, nil
}

func (s ProjectService) AddMember(projectId string, user domain.User) error {
	objID, err := primitive.ObjectIDFromHex(projectId)
	if err != nil {
		return fmt.Errorf("invalid project ID: %v", err)
	}

	return s.projects.AddMember(objID, user)
}

func (s *ProjectService) GetUser(username string) (domain.User, error) {

	url := fmt.Sprintf("http://user-server:8080/users/%s", username)

	resp, err := http.Get(url)
	if err != nil {
		return domain.User{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return domain.User{}, fmt.Errorf("user not found")
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return domain.User{}, err
	}

	var user domain.User
	if err := json.Unmarshal(body, &user); err != nil {
		return domain.User{}, fmt.Errorf("failed to decode user: %v", err)
	}

	// if user == nil {
	// 	return domain.User{}, fmt.Errorf("no user found")
	// }

	return user, nil
}

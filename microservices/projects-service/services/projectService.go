package services

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
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

func (s ProjectService) GetAll() (domain.Projects, error) {
	projects, err := s.projects.GetAll()
	if err != nil {
		return nil, fmt.Errorf("error fetching projects: %v", err)
	}
	return projects, nil
}

func (s *ProjectService) GetUser(username string) (domain.User, error) {
	log.Println("imeeeeee:", username)
	url := fmt.Sprintf("http://users-service:8000/api/users/users/%s", username)

	resp, err := http.Get(url)
	if err != nil {
		return domain.User{}, fmt.Errorf("failed to make request: %v", err)
	}
	defer resp.Body.Close()

	log.Printf("Response status: %d\n", resp.StatusCode)

	if resp.StatusCode == http.StatusNotFound {
		return domain.User{}, fmt.Errorf("user not found")
	} else if resp.StatusCode != http.StatusOK {
		return domain.User{}, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return domain.User{}, fmt.Errorf("failed to read response body: %v", err)
	}

	log.Println("Response body:", string(body))

	var user domain.User
	if err := json.Unmarshal(body, &user); err != nil {
		return domain.User{}, fmt.Errorf("failed to decode user: %v", err)
	}

	return user, nil
}

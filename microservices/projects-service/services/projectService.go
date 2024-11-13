package services

import (
	"encoding/json"
	"fmt"
	"log"
	"io"
	"net/http"
	"project-management-app/microservices/projects-service/domain"
	"project-management-app/microservices/projects-service/repositories"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ProjectService struct {
	projects *repositories.ProjectRepo
}

func NewUserService(p *repositories.ProjectRepo) *ProjectService {
	return &ProjectService{p}
}

func (s ProjectService) AddMember(projectId string, user domain.User) error {
	objID, err := primitive.ObjectIDFromHex(projectId)
	if err != nil {
		return fmt.Errorf("invalid project ID: %v", err)
	}

	return s.projects.AddMember(objID, user)
}

func (s *ProjectService) GetUser(username string) (domain.User, error) {

	url := fmt.Sprintf("http://users-service:8000/users/%s", username)

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

	body, err := io.ReadAll(resp.Body)
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

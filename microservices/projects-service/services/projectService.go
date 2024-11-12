package services

import (
	"encoding/json"
	"fmt"
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

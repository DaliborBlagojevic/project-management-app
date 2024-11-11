package services

import (
	"fmt"
	"log"
	"project-management-app/microservices/users-service/domain"
	"project-management-app/microservices/users-service/repositories"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
)

type UserService struct {
	users *repositories.UserRepo
}

func NewUserService(r *repositories.UserRepo) *UserService {
	return &UserService{r}
}

func (s UserService) Create(username, password, name, surname, email, roleString, activationCode string) (domain.User, error) {
	role, err := domain.RoleFromString(roleString)
	if err != nil {
		return domain.User{}, err
	}

	// Proveri da li postoji korisnik sa istim username-om
	existingUser, err := s.users.GetByUsername(username)
	if err != nil && err != mongo.ErrNoDocuments { // mongo.ErrNoDocuments znači da korisnik ne postoji
		return domain.User{}, err
	}
	if existingUser != nil {
		return domain.User{}, fmt.Errorf("user with username '%s' already exists", username)
	}

	// Kreiraj novog korisnika
	user := domain.User{
		Username:       username,
		Password:       password,
		Name:           name,
		Surname:        surname,
		Email:          email,
		Role:           role,
		IsActive:       false,
		ActivationCode: activationCode,
		CreatedAt:      time.Now(),
	}
	log.Println(user, "u servisu")

	return s.users.Insert(user)
}

func (s *UserService) GetAvailableMembers(projectId string) ([]map[string]interface{}, error) {
	return s.users.GetAvailableMembers(projectId)
}

func (s *UserService) PeriodicCleanup() {
	ticker := time.NewTicker(2 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			err := s.users.RemoveExpiredActivationCodes()
			if err != nil {
				log.Printf("Error during cleanup: %v", err)
			} else {
				log.Println("Successfully removed expired activation codes.")
			}
		}
	}
}

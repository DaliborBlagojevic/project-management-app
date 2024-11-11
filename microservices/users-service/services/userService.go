package services

import (

	"log"
	"example.com/project-management-app/microservices/users-service/domain"
	"example.com/project-management-app/microservices/users-service/repositories"
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

	existingUser, err := s.users.GetByUsername(username)
	if err != nil && err != mongo.ErrNoDocuments {
		return domain.User{}, err
	}
	if existingUser != nil {
		return domain.User{}, domain.ErrUserAlreadyExists()
	}

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

	return s.users.Insert(user)
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


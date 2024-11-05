package services

import (
	"fmt"
	"project-management-app/microservices/users-service/domain"
	"project-management-app/microservices/users-service/repositories"

	"go.mongodb.org/mongo-driver/mongo"
)

type UserService struct {

	users *repositories.UserRepo

}



func NewUserService( r *repositories.UserRepo) *UserService {
	return &UserService{r}
}



func (s UserService) Create(username, password, name, surname, email, roleString string) (domain.User, error) {
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

    // Proveri da li postoji korisnik sa istim email-om
    existingUser, err = s.users.GetByEmail(email)
    if err != nil && err != mongo.ErrNoDocuments {
        return domain.User{}, err
    }
    if existingUser != nil {
        return domain.User{}, fmt.Errorf("user with email '%s' already exists", email)
    }

    // Kreiraj novog korisnika
    user := domain.User{
        Username: username,
        Password: password,
        Name:     name,
        Surname:  surname,
        Email:    email,
        Role:     role,
    }

    return s.users.Insert(user)
}





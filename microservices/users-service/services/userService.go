package services

import (
	"project-management-app/microservices/users-service/domain"
)

type UserService struct {

	users domain.UserRepository

}

func NewUserService(users domain.UserRepository) (UserService, error) {
	return UserService{
		users: users,
	}, nil
}

func (s UserService) Create(username, password, name, surname,email string) (domain.User, error) {
	user := domain.User{
			Username: username,
			Password: password,
			Name: name,
			Surname: surname,
			Email: email,
	}
	return s.users.Create(user)
}

func (s UserService) LogIn(username, password string) (token string, err error) {
	users, err := s.users.GetAll()
	if err != nil {
		return
	}
	for _, user := range users {
		if user.Username == username && user.Password == password {
			token = user.Id.String()
			return
		}
	}
	err = domain.ErrInvalidCredentials()
	return
}
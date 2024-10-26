package services

import "project-management-app/microservices/users-service/domain"

type AuthService struct {
	users domain.UserRepository
}

func NewAuthService(users domain.UserRepository) (AuthService, error) {
	return AuthService{
		users: users,
	}, nil
}

func (s AuthService) LogIn(username, password string) (token string, err error) {
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

func (s AuthService) ResolveUser(token string) (authenticated domain.User, err error) {
	users, err := s.users.GetAll()
	if err != nil {
		return
	}
	for _, user := range users {
		if user.Id.String() == token {
			authenticated = user
			return
		}
	}
	err = domain.ErrInvalidToken()
	return
}

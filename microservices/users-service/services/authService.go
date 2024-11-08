package services

import (
	"project-management-app/microservices/users-service/domain"
	"project-management-app/microservices/users-service/repositories"
)



type AuthService struct {

	users *repositories.UserRepo

}



func NewAuthService( r *repositories.UserRepo) *AuthService {
	return &AuthService{r}
}




func (s AuthService) LogIn(username, password string) (token string, err error) {
	users, err := s.users.GetAllByUsername(username)
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

func (s AuthService) ResolveUser(token string) (authenticated *domain.User, err error) {
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

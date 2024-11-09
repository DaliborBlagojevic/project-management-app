package services

import (
	"fmt"
	"project-management-app/microservices/users-service/domain"
	"project-management-app/microservices/users-service/repositories"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var secretKey = []byte("bd7c66e8e9c7836c02a017e24a3f322a1a5b807d19f8f144b0da0469ca0353c40a4c4158fdb140d38ebf40e8e3153ab467e855a67a72916ce8f489b153b03f806d5ea225a6eec5b7feba06c2a0d78f70ae70a6c21ddd91e43c20a9582f7e8fbf97441827fe3ebc7029acc7f09ea218356114cabd1868215e609894d7163ead3070823e6b545963f7542761e4671e806dbe0139eb92458df251ea8f2bae810bf8494685c0f631abd6b651d231c98df5898ece84b451781816ce9bb49cc312f6d8a12fdb19357a73f0c50c48a071033168e5d6e498fa8668cbb13025edc1f2fea9862bfe2244b02ed3fef4eca27eba4d369d3ef17b855980b36686f4f40793904d")

type AuthService struct {
	users *repositories.UserRepo
}

func NewAuthService(r *repositories.UserRepo) *AuthService {
	return &AuthService{r}
}

func (s AuthService) LogIn(username, password string) (token string, err error) {
	var user *domain.User

	user, err = s.users.GetByUsername(username)
	if err != nil {
		return
	}

	if user.Password == password {
		return createToken(*user)
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

func createToken(user domain.User) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256,
		jwt.MapClaims{
			"username": user.Username,
			"name":     user.Name,
			"surname":  user.Surname,
			"email":    user.Email,
			"role":     user.Role,
			"exp":      time.Now().Add(time.Hour * 24).Unix(),
		})

	tokenString, err := token.SignedString(secretKey)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func verifyToken(tokenString string) error {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return secretKey, nil
	})

	if err != nil {
		return err
	}

	if !token.Valid {
		return fmt.Errorf("invalid token")
	}

	return nil
}

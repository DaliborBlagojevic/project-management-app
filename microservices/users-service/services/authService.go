package services

import (
	"fmt"
	"project-management-app/microservices/users-service/domain"
	"project-management-app/microservices/users-service/repositories"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type TokenClaims struct {
	Username string `json:"username"`
	Name     string `json:"name"`
	Surname  string `json:"surname"`
	Email    string `json:"email"`
	Role     string `json:"role"`
	Exp      int64  `json:"exp"`
}

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
		err = domain.ErrUserNotFound()
		return
	}

	if !user.IsActive {
		err = domain.ErrUserNotActive()
		return
	}

	if CheckPasswordHash(password, user.Password) {
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
			"role":     user.Role.String(),
			"exp":      time.Now().Add(time.Hour * 24).Unix(),
		})

	tokenString, err := token.SignedString(secretKey)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func (s AuthService) VerifyToken(tokenString string) (*TokenClaims, error) {
	// Parse the token with claims
	token, err := jwt.ParseWithClaims(tokenString, &jwt.MapClaims{}, func(token *jwt.Token) (interface{}, error) {
		return secretKey, nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	claims, ok := token.Claims.(*jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("unable to parse token claims")
	}

	tokenClaims := &TokenClaims{}

	if username, ok := (*claims)["username"].(string); ok {
		tokenClaims.Username = username
	}
	if name, ok := (*claims)["name"].(string); ok {
		tokenClaims.Name = name
	}
	if surname, ok := (*claims)["surname"].(string); ok {
		tokenClaims.Surname = surname
	}
	if email, ok := (*claims)["email"].(string); ok {
		tokenClaims.Email = email
	}
	if role, ok := (*claims)["role"].(string); ok {
		tokenClaims.Role = role
	}
	if exp, ok := (*claims)["exp"].(float64); ok {
		tokenClaims.Exp = int64(exp)
	}

	return tokenClaims, nil
}

func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

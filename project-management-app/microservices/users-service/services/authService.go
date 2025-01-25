package services

import (
	"context"
	"fmt"
	"os"
	"project-management-app/microservices/users-service/domain"
	"project-management-app/microservices/users-service/repositories"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
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

var secretKey = []byte(os.Getenv("SECRET_KEY_AUTH"))

type AuthService struct {
	users *repositories.UserRepo
	tracer trace.Tracer
}

func NewAuthService(r *repositories.UserRepo,t trace.Tracer) *AuthService {
	return &AuthService{r, t}
}

func (s AuthService) LogIn(ctx context.Context,username string, password string) (token string, err error) {
	ctx, span := s.tracer.Start(ctx, "Auth.LogIn")
    defer span.End()
	var user *domain.User

	user, err = s.users.GetByUsername(username)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		err = domain.ErrUserNotFound()
		return
	}

	if !user.IsActive {
		span.SetStatus(codes.Error, err.Error())
		err = domain.ErrUserNotActive()
		return
	}

	if CheckPasswordHash(password, user.Password) {
		return CreateToken(*user)
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

func CreateToken(user domain.User) (string, error) {
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

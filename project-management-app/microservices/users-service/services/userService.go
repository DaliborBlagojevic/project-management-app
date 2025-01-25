package services

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"project-management-app/microservices/users-service/domain"
	"project-management-app/microservices/users-service/repositories"
	"time"

	// "github.com/eapache/go-resiliency/retrier"
	"github.com/sony/gobreaker/v2"
	"go.mongodb.org/mongo-driver/mongo"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/crypto/bcrypt"
)

type UserService struct {
	users  *repositories.UserRepo
	cb     *gobreaker.CircuitBreaker[interface{}]
	client *http.Client
		tracer trace.Tracer
	projectServiceAddress string
}

func NewUserService(r *repositories.UserRepo) *UserService {
	cb := gobreaker.NewCircuitBreaker[interface{}](gobreaker.Settings{
		Name:        "UserServiceCB",
		MaxRequests: 1,
		Timeout:     2 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures > 0
		},
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			log.Printf("Circuit Breaker '%s' changed from '%s' to '%s'\n", name, from, to)
		},
	})

	client := &http.Client{
		Timeout: 5 * time.Second, // Globalni timeout
	}

	return &UserService{users: r, cb: cb, client: client, tracer :tracer}
}

func (s UserService) Create(ctx context.Context, username, password, name, surname, email, roleString, activationCode string) (domain.User, error) {
	ctx, span := s.tracer.Start(ctx, "UserService.Create")
    defer span.End()
	
	role, err := domain.RoleFromString(roleString)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return domain.User{}, err
	}

	existingUser, err := s.users.GetByUsername(username)
	if err != nil && err != mongo.ErrNoDocuments {
		span.SetStatus(codes.Error, err.Error())
		return domain.User{}, err
	}
	if existingUser != nil {
		span.SetStatus(codes.Error, err.Error())
		return domain.User{}, fmt.Errorf("user with username '%s' already exists", username)
	}
	hashedPassword, err := HashPassword(password)
	if err != nil {
		return domain.User{}, err
	}
	// Kreiraj novog korisnika
	user := domain.User{
		Username:       username,
		Password:       hashedPassword,
		Name:           name,
		Surname:        surname,
		Email:          email,
		Role:           role,
		IsActive:       false,
		ActivationCode: activationCode,
		CreatedAt:      time.Now(),
		IsExpired:      false,
	}

	return s.users.Insert(ctx, user)
}

func (s *UserService) PeriodicCleanup() {
	ticker := time.NewTicker(2 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		err := s.users.MarkExpiredActivationCodes()
		if err != nil {
			log.Printf("Error during cleanup: %v", err)
		} else {
			log.Println("Successfully removed expired activation codes.")
		}
	}
}

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

func (s *UserService) GetAvailableMembers(projectId string, projectMembers []domain.User) ([]domain.User, error) {
	return s.users.GetAvailableMembers(projectId, projectMembers)
}

func (s *UserService) ChangePassword(ctx context.Context, username string, oldPassword string, newPassword string, user domain.User) error {
	ctx, span := s.tracer.Start(ctx, "UserService.ChangePassword")
    defer span.End()
	// Proveri da li se stari password podudara sa postojećim passwordom iz baze
	err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(oldPassword))
	if err != nil {
		return errors.New("old password does not match the current password")
	}

	// Hesiraj novi password
	hashedNewPassword, err := HashPassword(newPassword)
	if err != nil {
		return err
	}

	// Pozovi funkciju za promenu passworda u repozitorijumu
	return s.users.ChangePassword(ctx, username, hashedNewPassword, &user)
}

func (s *UserService) RecoveryPassword(uuid string, password string, user *domain.User) error {
	hashedOldPassword, err := HashPassword(password)
	if err != nil {
		return err
	}
	return s.users.RecoveryPassword(uuid, hashedOldPassword, user)
}

func (us *UserService) Delete(ctx context.Context, user *domain.User) error {
    ctx, span := us.tracer.Start(ctx, "UserService.Delete")
    defer span.End()

    userID := user.Username

    if user.Role == domain.PROJECT_MANAGER {
        url := fmt.Sprintf("http://projects-service:8000/projects/manager/%s", userID)
        req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
        if err != nil {
            span.RecordError(err)
            span.SetStatus(codes.Error, err.Error())
            return fmt.Errorf("error creating request: %v", err)
        }

        // Inject tracing headers
        otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(req.Header))

        resp, err := http.DefaultClient.Do(req)
        if err != nil {
            span.RecordError(err)
            span.SetStatus(codes.Error, err.Error())
            return fmt.Errorf("error contacting projects-service: %v", err)
        }
        defer resp.Body.Close()

        if resp.StatusCode == http.StatusOK {
            return fmt.Errorf("cannot delete user: PROJECT_MANAGER is assigned to projects")
        } else if resp.StatusCode != http.StatusNoContent {
            return fmt.Errorf("unexpected response from projects-service: %v", resp.Status)
        }

    } else if user.Role == domain.PROJECT_MEMBER {
        url := fmt.Sprintf("http://projects-service:8000/projects/member/%s", userID)
        req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
        if err != nil {
            span.RecordError(err)
            span.SetStatus(codes.Error, err.Error())
            return fmt.Errorf("error creating request: %v", err)
        }

        // Inject tracing headers
        otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(req.Header))

        resp, err := http.DefaultClient.Do(req)
        if err != nil {
            span.RecordError(err)
            span.SetStatus(codes.Error, err.Error())
            return fmt.Errorf("error contacting projects-service: %v", err)
        }
        defer resp.Body.Close()

        if resp.StatusCode == http.StatusOK {
            return fmt.Errorf("cannot delete user: PROJECT_MEMBER is assigned to a project")
        } else if resp.StatusCode != http.StatusNoContent {
            return fmt.Errorf("unexpected response from projects-service: %v", resp.Status)
        }
    } else {
        err := fmt.Errorf("role not supported for deletion")
        span.RecordError(err)
        span.SetStatus(codes.Error, err.Error())
        return err
    }

    err := us.users.Delete(userID)
    if err != nil {
        span.RecordError(err)
        span.SetStatus(codes.Error, err.Error())
        return fmt.Errorf("error deleting user: %v", err)
    }

    span.SetStatus(codes.Ok, "")
    return nil
}


package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"project-management-app/microservices/projects-service/domain"
	"project-management-app/microservices/projects-service/repositories"
	"time"

	"github.com/eapache/go-resiliency/retrier"
	"github.com/sony/gobreaker/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.opentelemetry.io/otel/trace"
)

type ProjectService struct {
	projects *repositories.ProjectRepo
	cb       *gobreaker.CircuitBreaker[interface{}]
	client   *http.Client
	tracer trace.Tracer
}

func NewProjectService(p *repositories.ProjectRepo) *ProjectService {
	cb := gobreaker.NewCircuitBreaker[interface{}](gobreaker.Settings{
		Name:        "ProjectServiceCB",
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

	return &ProjectService{projects: p, cb: cb, client: client}
}

func (s ProjectService) AddMember(projectId string, user domain.User) error {
	objID, err := primitive.ObjectIDFromHex(projectId)
	if err != nil {
		return fmt.Errorf("invalid project ID: %v", err)
	}

	if err := s.sendNotification(user.Username, "You are added to project "); err != nil {
		fmt.Printf("Error sending notification: %v\n", err) // Dodato logovanje greške
		return fmt.Errorf("failed to send notification: %w", err)
	}

	return s.projects.AddMember(objID, user)
}

func (s ProjectService) RemoveMember(projectId string, username string) error {
	objID, err := primitive.ObjectIDFromHex(projectId)
	if err != nil {
		return fmt.Errorf("invalid project ID: %v", err)
	}

	if err := s.sendNotification(username, "You are deleted from project"); err != nil {
		fmt.Printf("Error sending notification: %v\n", err) // Dodato logovanje greške
		return fmt.Errorf("failed to send notification: %w", err)
	}

	return s.projects.RemoveMember(objID, username)
}

func (s *ProjectService) GetUser(username string) (domain.User, error) {
	url := fmt.Sprintf("http://users-service:8000/users/%s", username)

	r := retrier.New(retrier.ConstantBackoff(3, 100*time.Millisecond), nil)

	var user domain.User
	err := r.Run(func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			log.Println("Error creating request:", err)
			return fmt.Errorf("failed to create request: %v", err)
		}

		resp, err := s.client.Do(req)
		if err != nil {
			log.Println("Error making request to user service:", err)
			return fmt.Errorf("failed to get user: %v", err)
		}
		defer resp.Body.Close()

		log.Printf("Response status: %d\n", resp.StatusCode)

		if resp.StatusCode == http.StatusNotFound {
			log.Println("User not found")
			return fmt.Errorf("user not found")
		} else if resp.StatusCode != http.StatusOK {
			log.Printf("Unexpected status code: %d\n", resp.StatusCode)
			return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Println("Error reading response body:", err)
			return fmt.Errorf("failed to read response body: %v", err)
		}

		log.Println("Response body:", string(body))

		if err := json.Unmarshal(body, &user); err != nil {
			log.Println("Failed to decode user:", err)
			return fmt.Errorf("failed to decode user: %v", err)
		}

		return nil
	})

	if err != nil {
		return domain.User{}, err
	}

	return user, nil
}

func (s *ProjectService) GetProjectsByManager(ctx context.Context,username string) (domain.Projects, error) {
	ctx, span := s.tracer.Start(ctx, "ProductRepository.GetProjectsByManager")
	defer span.End()
	return s.projects.GetProjectsByManager(username)
}

func (ps *ProjectService) GetProjectsByMember(ctx context.Context,username string) (domain.Projects, error) {
	ctx, span := ps.tracer.Start(ctx, "ProductRepository.GetProjectsByMember")
	defer span.End()
	return ps.projects.GetProjectsByMember(username)
}

func (s ProjectService) sendNotification(username, message string) error {
	url := "http://notifications-service:8000/notifications"

	payload := map[string]string{
		"user_id": username,
		"message": message,
	}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		log.Println("Failed to marshal payload:", err)
		return fmt.Errorf("failed to send notification: %v", err)
	}

	r := retrier.New(retrier.ConstantBackoff(3, 100*time.Millisecond), nil)

	err = r.Run(func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonData))
		if err != nil {
			log.Println("Error creating request:", err)
			return fmt.Errorf("failed to create request: %v", err)
		}

		req.Header.Set("Content-Type", "application/json")

		resp, err := s.client.Do(req)
		if err != nil {
			log.Println("Failed to send notification:", err)
			return fmt.Errorf("failed to send notification: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
			body, _ := io.ReadAll(resp.Body)
			log.Printf("Failed to send notification: status %d, response: %s", resp.StatusCode, string(body))
			return fmt.Errorf("failed to send notification: unexpected status code %d", resp.StatusCode)
		}

		return nil
	})

	return err
}

func (s *ProjectService) GetProjectsByManagerAndIsActive(ctx context.Context, username string) (domain.Projects, error) {
	projects, err := s.projects.GetProjectsByManagerAndIsActive(ctx, username)
	if err != nil {
		return nil, err
	}

	// Filtriraj projekte prema isActive
	var filteredProjects domain.Projects
	for _, project := range projects {
		if !project.IsActive {
			filteredProjects = append(filteredProjects, project)
		}
	}

	return filteredProjects, nil
}

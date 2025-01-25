package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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

type TaskService struct {
	tasks  *repositories.TaskRepo
	cb     *gobreaker.CircuitBreaker[interface{}]
	
	client *http.Client
	tracer trace.Tracer
}

func NewTaskService(tasks *repositories.TaskRepo) *TaskService {
	cb := gobreaker.NewCircuitBreaker[interface{}](gobreaker.Settings{
		Name:        "TaskServiceCB",
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

	return &TaskService{tasks: tasks, cb: cb, client: client, tracer: tracer}
}

func (s TaskService) AddMember(taskId string, user domain.User) error {
	task, err := s.tasks.FindById(taskId)
	if err != nil {
		return err
	}

	projectMembers, err := s.getProjectMembers(task.Project)
	if err != nil {
		return err
	}

	isMember := false
	for _, member := range projectMembers {
		if member.Username == user.Username {
			isMember = true
			break
		}
	}

	if !isMember {
		return errors.New("user is not a member of the project")
	}

	// Pozivanje funkcije za slanje notifikacije
	if err := s.sendNotification(user.Username, "You are added to task "+task.Name); err != nil {
		fmt.Printf("Error sending notification: %v\n", err) // Dodato logovanje greške
		return fmt.Errorf("failed to send notification: %w", err)
	}

	return s.tasks.AddMember(task.Id, user)
}

// Funkcija za slanje notifikacije
func (s TaskService) sendNotification(username, message string) error {
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

		// Provera validnih statusnih kodova
		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
			body, _ := io.ReadAll(resp.Body)
			log.Printf("Failed to send notification: status %d, response: %s", resp.StatusCode, string(body))
			return fmt.Errorf("failed to send notification: unexpected status code %d", resp.StatusCode)
		}

		return nil
	})

	return err
}

// Create - Kreira novi zadatak sa prosleđenim parametrima
func (s TaskService) Create(ctx context.Context,status domain.Status, name string, description string, projectID string) (domain.Task, error) {

	ctx, span := s.tracer.Start(ctx, "TasksService.Create")
	defer span.End()
	
	existingTask, err := s.tasks.FindByName(name)
	if err != nil {
		return domain.Task{}, err
	}
	if existingTask != nil {
		return domain.Task{}, errors.New("zadatak sa istim imenom već postoji")
	}

	// Kreiraj novi zadatak
	task := domain.Task{
		Id:          primitive.NewObjectID(),
		Project:     projectID,
		Name:        name,
		Description: description,
		Status:      1,
	}

	return s.tasks.Insert(ctx, task)
}

func (s TaskService) Update(id string, status domain.Status, name string, description string, projectID string) (domain.Task, error) {
	existingTask, err := s.tasks.FindById(id)
	if err != nil {
		return domain.Task{}, errors.New("task doesn't exist")
	}

	existingTask.Name = name
	existingTask.Description = description
	existingTask.Status = status

	updatedTask, err := s.tasks.Update(*existingTask)
	for _, member := range updatedTask.Members {
		if err := s.sendNotification(member.Username, "Task "+updatedTask.Name+" updated"); err != nil {
			fmt.Printf("Error sending notification: %v\n", err)
			return domain.Task{}, fmt.Errorf("failed to send notification: %w", err)
		}
	}
	return updatedTask, err
}

func (s TaskService) FilterMembersNotOnTask(projectId, taskId string) (domain.Users, error) {
	// Dobavi članove sa projekta
	projectMembers, err := s.getProjectMembers(projectId)
	if err != nil {
		return nil, err
	}

	// Dobavi članove koji su već na zadatku
	taskMembers, err := s.tasks.GetMembersByTaskId(taskId)
	if err != nil {
		return nil, err
	}

	// Kreiraj mapu za brzu proveru članova koji su na zadatku
	taskMemberMap := make(map[string]struct{})
	for _, member := range taskMembers {
		taskMemberMap[member.Username] = struct{}{}
	}

	// Filtriraj članove sa projekta koji nisu na zadatku
	filteredMembers := domain.Users{}
	for _, member := range projectMembers {
		if _, exists := taskMemberMap[member.Username]; !exists {
			filteredMembers = append(filteredMembers, member)
		}
	}

	return filteredMembers, nil
}

func (s TaskService) getProjectMembers(projectId string) (domain.Users, error) {
	url := fmt.Sprintf("http://projects-service:8000/projects/members/%s", projectId)

	r := retrier.New(retrier.ConstantBackoff(3, 100*time.Millisecond), nil)

	var members domain.Users
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
			log.Println("Failed to fetch project members:", err)
			return fmt.Errorf("failed to fetch project members: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			log.Printf("Error fetching project members, status code: %d", resp.StatusCode)
			return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}

		if err := json.NewDecoder(resp.Body).Decode(&members); err != nil {
			log.Println("Failed to decode project members:", err)
			return fmt.Errorf("failed to decode project members: %v", err)
		}

		return nil
	})

	if err != nil {
		return domain.Users{}, err
	}

	return members, nil
}

func (s TaskService) RemoveMember(taskId string, user domain.User) error {
	objID, err := primitive.ObjectIDFromHex(taskId)
	task, err := s.tasks.FindById(taskId)
	if err != nil {
		return fmt.Errorf("invalid task ID: %v", err)
	}

	if err := s.sendNotification(user.Username, "You are deleted from task "+task.Name); err != nil {
		fmt.Printf("Error sending notification: %v\n", err) // Dodato logovanje greške
		return fmt.Errorf("failed to send notification: %w", err)
	}

	return s.tasks.RemoveMember(objID, user)
}

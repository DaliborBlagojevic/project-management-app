package repositories

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"project-management-app/microservices/projects-service/domain"
	"time"

	"github.com/eapache/go-resiliency/retrier"
	"github.com/sony/gobreaker/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.opentelemetry.io/otel/trace"
)

type ProjectRepo struct {
	cli    *mongo.Client
	logger *log.Logger
	tracer trace.Tracer
	cb     *gobreaker.CircuitBreaker[interface{}]
	client *http.Client
}

func New(ctx context.Context, logger *log.Logger, tracer trace.Tracer) (*ProjectRepo, error) {
	dburi := os.Getenv("MONGO_DB_URI")

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(dburi))
	if err != nil {
		return nil, err
	}

	cb := gobreaker.NewCircuitBreaker[interface{}](gobreaker.Settings{
		Name:        "ProjectRepoCB",
		MaxRequests: 1,
		Timeout:     2 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures > 0
		},
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			log.Printf("Circuit Breaker '%s' changed from '%s' to '%s'\n", name, from, to)
		},
	})

	httpClient := &http.Client{
		Timeout: 5 * time.Second, // Globalni timeout
	}

	return &ProjectRepo{
		cli:    client,
		logger: logger,
		tracer: tracer,
		cb:     cb,
		client: httpClient,
	}, nil
}

func (pr *ProjectRepo) Disconnect(ctx context.Context) error {
	err := pr.cli.Disconnect(ctx)
	if err != nil {
		return err
	}
	return nil
}

func (pr *ProjectRepo) Ping() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Check connection -> if no error, connection is established
	err := pr.cli.Ping(ctx, readpref.Primary())
	if err != nil {
		pr.logger.Println(err)
	}

	// Print available databases
	databases, err := pr.cli.ListDatabaseNames(ctx, bson.M{})
	if err != nil {
		pr.logger.Println(err)
	}
	fmt.Println(databases)
}

func (ur *ProjectRepo) getCollection() *mongo.Collection {
	projectDatabase := ur.cli.Database("projects")
	projectsCollection := projectDatabase.Collection("projects")
	return projectsCollection
}

func (pr *ProjectRepo) GetAll() (domain.Projects, error) {
	// Initialise context (after 5 seconds timeout, abort operation)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	projectsCollection := pr.getCollection()

	var projects domain.Projects
	projectsCursor, err := projectsCollection.Find(ctx, bson.M{})
	if err != nil {
		pr.logger.Println(err)
		return nil, err
	}
	if err = projectsCursor.All(ctx, &projects); err != nil {
		pr.logger.Println(err)
		return nil, err
	}
	return projects, nil
}

func (ur *ProjectRepo) AddMember(projectId primitive.ObjectID, user domain.User) error {
	// Send request to user microservice
	url := fmt.Sprintf("http://users-service:8000/projects/%s/availableMembers", projectId.Hex())
	reqBody, err := json.Marshal(map[string]string{
		"projectId": projectId.Hex(),
	})
	if err != nil {
		ur.logger.Println("Error marshalling request body:", err)
		return fmt.Errorf("failed to add member: %v", err)
	}

	r := retrier.New(retrier.ConstantBackoff(3, 100*time.Millisecond), nil)

	_, err = ur.cb.Execute(func() (interface{}, error) {
		err := r.Run(func() error {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(reqBody))
			if err != nil {
				ur.logger.Println("Error creating request:", err)
				return fmt.Errorf("failed to create request: %v", err)
			}

			req.Header.Set("Content-Type", "application/json")

			resp, err := ur.client.Do(req)
			if err != nil {
				ur.logger.Println("Error making request to user service:", err)
				return fmt.Errorf("failed to add member: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
				ur.logger.Println("Unexpected status code:", resp.StatusCode)
				return fmt.Errorf("failed to add member: unexpected status code %d", resp.StatusCode)
			}

			var availableMembers []domain.User
			if err := json.NewDecoder(resp.Body).Decode(&availableMembers); err != nil {
				ur.logger.Println("Error decoding response from user service:", err)
				return fmt.Errorf("failed to add member: %v", err)
			}

			// Check if user exists in the available members list
			userExists := false
			for _, availableUser := range availableMembers {
				if availableUser.Username == user.Username && availableUser.Name == user.Name && availableUser.Surname == user.Surname {
					userExists = true
					break
				}
			}

			if !userExists {
				ur.logger.Println("User not available")
				return fmt.Errorf("failed to add member: user not available")
			}

			// Check if user is already a member of the project
			ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			existsFilter := bson.M{
				"_id":              projectId,
				"members.username": user.Username,
				"members.name":     user.Name,
				"members.surname":  user.Surname,
			}
			count, err := ur.getCollection().CountDocuments(ctx, existsFilter)
			if err != nil {
				ur.logger.Println("Error checking existing members:", err)
				return fmt.Errorf("failed to add member: %v", err)
			}
			if count > 0 {
				ur.logger.Println("User already a member")
				return fmt.Errorf("failed to add member: user already a member")
			}

			// Add member to the project
			_, err = ur.getCollection().UpdateOne(
				ctx,
				bson.M{"_id": projectId},
				bson.M{
					"$push": bson.M{"members": user},
				},
			)
			if err != nil {
				ur.logger.Println("Error updating document:", err)
				return fmt.Errorf("failed to add member: %v", err)
			}

			return nil
		})

		return nil, err
	})

	return err
}

func (ur *ProjectRepo) RemoveMember(projectId primitive.ObjectID, username string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	projectsCollection := ur.getCollection()
	_, err := projectsCollection.UpdateOne(
		ctx,
		bson.M{"_id": projectId},
		bson.M{
			"$pull": bson.M{"members": bson.M{"username": username}},
		},
	)
	if err != nil {
		ur.logger.Println("Error updating document:", err)
		return err
	}

	return nil
}

func (ur *ProjectRepo) GetById(id string, username string, role string) (*domain.Project, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	projectsCollection := ur.getCollection()

	var project domain.Project
	objID, _ := primitive.ObjectIDFromHex(id)
	var err error
	switch role {
	case "PROJECT_MANAGER":
		err = projectsCollection.FindOne(ctx, bson.M{"_id": objID, "manager.username": username}).Decode(&project)
	case "PROJECT_MEMBER":
		err = projectsCollection.FindOne(ctx, bson.M{"_id": objID, "members.username": username}).Decode(&project)
	default:
		return nil, fmt.Errorf("invalid role provided")
	}

	if err == mongo.ErrNoDocuments {
		return nil, fmt.Errorf("user is not a member or manager of the project")
	} else if err != nil {
		ur.logger.Println(err)
		return nil, err
	}

	return &project, nil
}

func (ur *ProjectRepo) GetMembersByProjectId(ctx context.Context,id string) (domain.Users, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	projectsCollection := ur.getCollection()

	var project domain.Project
	objID, _ := primitive.ObjectIDFromHex(id)
	err := projectsCollection.FindOne(ctx, bson.M{"_id": objID}).Decode(&project)
	if err != nil {
		ur.logger.Println(err)
		return nil, err
	}

	if project.Members == nil {
		return domain.Users{}, nil
	}

	return project.Members, nil
}

func (pr *ProjectRepo) Create(ctx context.Context, project *domain.Project) error {
    ctx, span := pr.tracer.Start(ctx, "ProjectsRepo.Create")
    defer span.End()
	projectsCollection := pr.getCollection()

	result, err := projectsCollection.InsertOne(ctx, &project)
	if err != nil {
		pr.logger.Println(err)
		return err
	}
	pr.logger.Printf("Documents ID: %v\n", result.InsertedID)
	return nil
}

func (ur *ProjectRepo) GetProjectsByManager(username string) (domain.Projects, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	projectsCollection := ur.getCollection()

	var projects domain.Projects
	filter := bson.M{"manager.username": username}
	projectsCursor, err := projectsCollection.Find(ctx, filter)
	if err != nil {
		ur.logger.Println(err)
		return nil, err
	}
	if err = projectsCursor.All(ctx, &projects); err != nil {
		ur.logger.Println(err)
		return nil, err
	}
	return projects, nil
}

func (pr *ProjectRepo) GetProjectsByMember(username string) (domain.Projects, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	projectsCollection := pr.getCollection()

	var projects domain.Projects
	filter := bson.M{"members.username": username}
	projectsCursor, err := projectsCollection.Find(ctx, filter)
	if err != nil {
		pr.logger.Println(err)
		return nil, err
	}
	if err = projectsCursor.All(ctx, &projects); err != nil {
		pr.logger.Println(err)
		return nil, err
	}
	return projects, nil
}

func (pr *ProjectRepo) GetProjectsByManagerAndIsActive(ctx context.Context,username string) (domain.Projects, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	projectsCollection := pr.getCollection()

	var projects domain.Projects
	filter := bson.M{"manager.username": username}
	projectsCursor, err := projectsCollection.Find(ctx, filter)
	if err != nil {
		pr.logger.Println(err)
		return nil, err
	}
	if err = projectsCursor.All(ctx, &projects); err != nil {
		pr.logger.Println(err)
		return nil, err
	}
	return projects, nil
}

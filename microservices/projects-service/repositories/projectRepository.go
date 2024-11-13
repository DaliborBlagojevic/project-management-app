package repositories

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"project-management-app/microservices/projects-service/domain"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

type ProjectRepo struct {
	cli    *mongo.Client
	logger *log.Logger
}

func New(ctx context.Context, logger *log.Logger) (*ProjectRepo, error) {
	dburi := os.Getenv("MONGO_DB_URI")

	client, err := mongo.NewClient(options.Client().ApplyURI(dburi))
	if err != nil {
		return nil, err
	}

	err = client.Connect(ctx)
	if err != nil {
		return nil, err
	}

	return &ProjectRepo{
		cli:    client,
		logger: logger,
	}, nil
}

func (pr *ProjectRepo) Disconnect(ctx context.Context) error {
	err := pr.cli.Disconnect(ctx)
	if err != nil {
		return err
	}
	return nil
}

// Check database connection
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

func (ur *ProjectRepo) GetAllByManager(managerId string) (domain.Projects, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	projectsCollection := ur.getCollection()

	var projects domain.Projects
	projectsCursor, err := projectsCollection.Find(ctx, bson.M{"managerId": managerId})
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

func (ur *ProjectRepo) AddMember(projectId primitive.ObjectID, user domain.User) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	projectsCollection := ur.getCollection()
	_, err := projectsCollection.UpdateOne(
		ctx,
		bson.M{"_id": projectId},
		bson.M{
			"$push": bson.M{"members": user},
		},
	)
	if err != nil {
		ur.logger.Println("Error updating document:", err)
		return err
	}

	return nil
}

func (ur *ProjectRepo) GetById(id string) (*domain.Project, error) {
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

	return &project, nil
}

func (ur *ProjectRepo) getCollection() *mongo.Collection {
	projectDatabase := ur.cli.Database("projects")
	projectsCollection := projectDatabase.Collection("projects")
	return projectsCollection
}

func (pr *ProjectRepo) Create(project *domain.Project) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	projectsCollection := pr.getCollection()

	result, err := projectsCollection.InsertOne(ctx, &project)
	if err != nil {
		pr.logger.Println(err)
		return err
	}
	pr.logger.Printf("Documents ID: %v\n", result.InsertedID)
	return nil
}

func GetUserById(id string) (domain.User, error) {
	url := fmt.Sprintf("http://users-service:8000/users/id/%s", id)

	resp, err := http.Get(url)
	if err != nil {
		return domain.User{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return domain.User{}, err
	}

	// Pretpostavljamo da API vraća pojedinačnog korisnika kao JSON objekat
	var user domain.User
	if err := json.Unmarshal(body, &user); err != nil {
		return domain.User{}, fmt.Errorf("failed to decode user: %v", err)
	}

	return user, nil
}

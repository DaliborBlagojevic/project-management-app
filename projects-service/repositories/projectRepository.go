package repositories

import (
	"context"
	"fmt"
	"log"
	"os"
	"project-management-app/microservices/projects-service/domain"
	"project-management-app/microservices/projects-service/dto"
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

func (ur *ProjectRepo) GetAll() (domain.Projects, error) {
	// Initialise context (after 5 seconds timeout, abort operation)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	usersCollection := ur.getCollection()

	var users domain.Projects
	usersCursor, err := usersCollection.Find(ctx, bson.M{})
	if err != nil {
		ur.logger.Println(err)
		return nil, err
	}
	if err = usersCursor.All(ctx, &users); err != nil {
		ur.logger.Println(err)
		return nil, err
	}
	return users, nil
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
	projectDatabase := ur.cli.Database("mongoDemo")
	projectsCollection := projectDatabase.Collection("projects")
	return projectsCollection
}

func (ur *ProjectRepo) Insert(project domain.Project) (domain.Project, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	projectDto := dto.NewProjectDto(project)

	projectsCollection := ur.getCollection()
	result, err := projectsCollection.InsertOne(ctx, &projectDto)
	if err != nil {
		ur.logger.Println("Error inserting document:", err)
		return domain.Project{}, err
	}

	ur.logger.Printf("Document ID: %v\n", result.InsertedID)
	return project, nil
}

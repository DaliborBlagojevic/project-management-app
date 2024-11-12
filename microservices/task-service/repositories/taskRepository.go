package repositories

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"project-management-app/microservices/projects-service/domain"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

type TaskRepo struct {
	cli        *mongo.Client
	collection *mongo.Collection
	logger     *log.Logger
}

// NewTaskRepo kreira novu instancu TaskRepo i povezuje se sa MongoDB bazom
func NewTaskRepo(ctx context.Context, logger *log.Logger) (*TaskRepo, error) {
	dbURI := os.Getenv("MONGO_DB_URI")
	if dbURI == "" {
		return nil, fmt.Errorf("MONGO_DB_URI nije postavljen")
	}

	// Kreiramo klijenta za MongoDB sa prosleđenim URI-jem
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(dbURI))
	if err != nil {
		return nil, fmt.Errorf("nije moguće povezati se na MongoDB: %w", err)
	}

	// Proveravamo konekciju sa bazom
	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		return nil, fmt.Errorf("nije moguće ostvariti konekciju sa MongoDB: %w", err)
	}

	// Kreiramo kolekciju "tasks" unutar baze "project-management"
	taskCollection := client.Database("project-management").Collection("tasks")

	logger.Println("Povezan sa MongoDB")

	return &TaskRepo{
		cli:        client,
		collection: taskCollection,
		logger:     logger,
	}, nil
}

// Insert umeće novi zadatak u MongoDB kolekciju
func (pr *TaskRepo) Insert(task domain.Task) (domain.Task, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := pr.collection.InsertOne(ctx, task)
	if err != nil {
		pr.logger.Println("Greška prilikom umetanja zadatka:", err)
		return domain.Task{}, err
	}

	pr.logger.Println("Zadatak uspešno umetnut:", task.Id)
	return task, nil
}

// Disconnect zatvara konekciju sa MongoDB
func (pr *TaskRepo) Disconnect(ctx context.Context) error {
	err := pr.cli.Disconnect(ctx)
	if err != nil {
		return fmt.Errorf("greška prilikom zatvaranja konekcije: %w", err)
	}
	pr.logger.Println("Konekcija sa MongoDB zatvorena")
	return nil
}

// Ping proverava konekciju sa bazom i ispisuje dostupne baze
func (pr *TaskRepo) Ping(ctx context.Context) error {
	err := pr.cli.Ping(ctx, readpref.Primary())
	if err != nil {
		pr.logger.Println("Greška prilikom povezivanja sa bazom:", err)
		return err
	}

	// Ispis dostupnih baza podataka
	databases, err := pr.cli.ListDatabaseNames(ctx, bson.M{})
	if err != nil {
		pr.logger.Println("Greška prilikom listanja baza:", err)
		return err
	}
	fmt.Println("Dostupne baze podataka:", databases)
	return nil
}

package repositories

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"project-management-app/microservices/projects-service/domain"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.opentelemetry.io/otel/trace"
)

type TaskRepo struct {
	cli    *mongo.Client
	logger *log.Logger
	tracer trace.Tracer
}

// NewTaskRepo kreira novu instancu TaskRepo i povezuje se sa MongoDB bazom
func NewTaskRepo(ctx context.Context, logger *log.Logger,tracer trace.Tracer) (*TaskRepo, error) {
	dburi := os.Getenv("MONGO_DB_URI")

	client, err := mongo.NewClient(options.Client().ApplyURI(dburi))
	if err != nil {
		return nil, err
	}

	err = client.Connect(ctx)
	if err != nil {
		return nil, err
	}

	return &TaskRepo{
		cli:    client,
		logger: logger,
		tracer: tracer,
	}, nil
}

func (pr *TaskRepo) Disconnect(ctx context.Context) error {
	err := pr.cli.Disconnect(ctx)
	if err != nil {
		return err
	}
	return nil
}

// Check database connection
func (pr *TaskRepo) Ping() {
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

func (ur *TaskRepo) getCollection() *mongo.Collection {
	TaskDatabase := ur.cli.Database("tasks")
	tasksCollection := TaskDatabase.Collection("tasks")
	return tasksCollection
}

func (pr *TaskRepo) FindByName(name string) (*domain.Task, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tasksCollection := pr.getCollection()

	var task domain.Task
	filter := bson.M{"name": name}
	err := tasksCollection.FindOne(ctx, filter).Decode(&task)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil // Vraća nil ako nema rezultata
		}
		pr.logger.Println("Greška prilikom traženja zadatka:", err)
		return nil, err
	}

	return &task, nil
}

func (pr *TaskRepo) GetByProject(ctx context.Context ,projectId string) (domain.Tasks, error) {
	ctx, span := pr.tracer.Start(ctx, "Tasks.Handler.GetByProject")
	defer span.End()

	patientsCollection := pr.getCollection()

	var tasks domain.Tasks
	patientsCursor, err := patientsCollection.Find(ctx, bson.M{"project": projectId})
	if err != nil {
		pr.logger.Println(err)
		return nil, err
	}
	if err = patientsCursor.All(ctx, &tasks); err != nil {
		pr.logger.Println(err)
		return nil, err
	}
	return tasks, nil
}

func (pr *TaskRepo) FindByProject(projectId string) (*domain.Task, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tasksCollection := pr.getCollection()

	var task domain.Task
	filter := bson.M{"project": projectId}
	err := tasksCollection.FindOne(ctx, filter).Decode(&task)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil // Vraća nil ako nema rezultata
		}
		pr.logger.Println("Greška prilikom traženja zadatka:", err)
		return nil, err
	}

	return &task, nil
}

func (pr *TaskRepo) GetAll() (domain.Tasks, error) {
	// Initialise context (after 5 seconds timeout, abort operation)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tasksCollection := pr.getCollection()

	var tasks domain.Tasks
	tasksCursor, err := tasksCollection.Find(ctx, bson.M{})
	if err != nil {
		pr.logger.Println(err)
		return nil, err
	}
	if err = tasksCursor.All(ctx, &tasks); err != nil {
		pr.logger.Println(err)
		return nil, err
	}
	return tasks, nil
}

func (ur *TaskRepo) GetMembersByTaskId(id string) (domain.Users, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tasksCollection := ur.getCollection()

	var task domain.Task
	objID, _ := primitive.ObjectIDFromHex(id)
	err := tasksCollection.FindOne(ctx, bson.M{"_id": objID}).Decode(&task)
	if err != nil {
		ur.logger.Println(err)
		return nil, err
	}

	return task.Members, nil
}

func (ur *TaskRepo) GetAllByManager(id string) (domain.Tasks, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tasksCollection := ur.getCollection()

	var projects domain.Tasks
	projectsCursor, err := tasksCollection.Find(ctx, bson.M{"project": id})
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

func (pr *TaskRepo) FindByProjectId(projectId string) (*domain.Tasks, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tasksCollection := pr.getCollection()
	log.Println(projectId)
	var tasks domain.Tasks
	filter := bson.M{"project": projectId}
	err := tasksCollection.FindOne(ctx, filter).Decode(&tasks)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil // Vraća nil ako nema rezultata
		}
		pr.logger.Println("Greška prilikom traženja zadatka:", err)
		return nil, err
	}

	return &tasks, nil
}

func (ur *TaskRepo) AddMember(projectId primitive.ObjectID, user domain.User) error {
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

func (ur *TaskRepo) FindById(id string) (*domain.Task, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tasksCollection := ur.getCollection()

	var task domain.Task
	objID, _ := primitive.ObjectIDFromHex(id)
	err := tasksCollection.FindOne(ctx, bson.M{"_id": objID}).Decode(&task)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		ur.logger.Println(err)
		return nil, err
	}

	return &task, nil
}

func (ur *TaskRepo) RemoveMember(taskId primitive.ObjectID, user domain.User) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tasksCollection := ur.getCollection()
	_, err := tasksCollection.UpdateOne(
		ctx,
		bson.M{"_id": taskId},
		bson.M{
			"$pull": bson.M{"members": bson.M{"username": user.Username}},
		},
	)
	if err != nil {
		ur.logger.Println("Error updating document:", err)
		return err
	}

	return nil
}

// Insert umeće novi zadatak u MongoDB kolekciju
func (pr *TaskRepo) Insert(ctx context.Context, task domain.Task) (domain.Task, error) {
	ctx, span := pr.tracer.Start(ctx, "Tasks.Handler.GetByProject")
	defer span.End()

	tasksCollection := pr.getCollection()

	_, err := tasksCollection.InsertOne(ctx, task)
	if err != nil {
		pr.logger.Println("Greška prilikom umetanja zadatka:", err)
		return domain.Task{}, err
	}

	pr.logger.Println("Zadatak uspešno umetnut:", task.Id)
	return task, nil
}

func (pr *TaskRepo) Update(updatedTask domain.Task) (domain.Task, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tasksCollection := pr.getCollection()

	filter := bson.M{"_id": updatedTask.Id}
	update := bson.M{
		"$set": bson.M{
			"project":     updatedTask.Project,
			"name":        updatedTask.Name,
			"description": updatedTask.Description,
			"status":      updatedTask.Status,
		},
	}

	_, err := tasksCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		pr.logger.Println("Error updating task:", err)
		return domain.Task{}, fmt.Errorf("failed to update task: %w", err)
	}

	var task domain.Task
	err = tasksCollection.FindOne(ctx, filter).Decode(&task)
	if err != nil {
		pr.logger.Println("Error fetching updated task:", err)
		return domain.Task{}, fmt.Errorf("failed to fetch updated task: %w", err)
	}

	pr.logger.Println("Task successfully updated:", task.Id)
	return task, nil
}

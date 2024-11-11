package repositories

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"project-management-app/microservices/projects-service/dao"
	"project-management-app/microservices/projects-service/domain"
	"strconv"
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
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	projectsCollection := pr.getCollection()

	var rawProjects []map[string]interface{}
	projectsCursor, err := projectsCollection.Find(ctx, bson.M{})
	if err != nil {
		pr.logger.Println("Error fetching projects:", err)
		return nil, err
	}
	if err = projectsCursor.All(ctx, &rawProjects); err != nil {
		pr.logger.Println("Error decoding projects:", err)
		return nil, err
	}

	var projects domain.Projects
	for _, rawProject := range rawProjects {
		managerID, ok := rawProject["manager"].(primitive.ObjectID)
		if !ok {
			pr.logger.Println("Manager ID is not an ObjectID for project:", rawProject["name"])
			continue
		}

		manager, err := GetUserById(managerID.Hex())
		if err != nil {
			pr.logger.Println("Error fetching manager for project:", rawProject["name"], err)
			continue
		}

		// Provera i konverzija `enddate` polja
		var endDate time.Time
		if rawEndDate, exists := rawProject["enddate"]; exists && rawEndDate != nil {
			endDateStr, ok := rawEndDate.(string)
			if !ok {
				pr.logger.Println("Error: enddate is not a string for project:", rawProject["name"])
				continue
			}
			endDate, err = time.Parse("2006-01-02", endDateStr)
			if err != nil {
				pr.logger.Println("Error parsing enddate for project:", rawProject["name"], err)
				continue
			}
		}

		// Konverzija `minworkers` i `maxworkers` polja iz stringa u int32
		var minWorkers, maxWorkers int32
		if rawMinWorkers, ok := rawProject["minworkers"].(string); ok {
			parsedMinWorkers, err := strconv.Atoi(rawMinWorkers)
			if err != nil {
				pr.logger.Println("Error parsing minworkers for project:", rawProject["name"], err)
				continue
			}
			minWorkers = int32(parsedMinWorkers)
		} else if rawMinWorkers, ok := rawProject["minworkers"].(int32); ok {
			minWorkers = rawMinWorkers
		}

		if rawMaxWorkers, ok := rawProject["maxworkers"].(string); ok {
			parsedMaxWorkers, err := strconv.Atoi(rawMaxWorkers)
			if err != nil {
				pr.logger.Println("Error parsing maxworkers for project:", rawProject["name"], err)
				continue
			}
			maxWorkers = int32(parsedMaxWorkers)
		} else if rawMaxWorkers, ok := rawProject["maxworkers"].(int32); ok {
			maxWorkers = rawMaxWorkers
		}

		project := &domain.Project{
			Id:         rawProject["_id"].(primitive.ObjectID),
			Manager:    manager,
			Name:       rawProject["name"].(string),
			EndDate:    endDate,
			MinWorkers: int(minWorkers),
			MaxWorkers: int(maxWorkers),
		}

		projects = append(projects, project)
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

func (ur *ProjectRepo) Update(project domain.Project) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	projectDto := dao.NewProjectDao(project)

	projectsCollection := ur.getCollection()
	_, err := projectsCollection.ReplaceOne(ctx, bson.M{"_id": project.Id}, &projectDto)
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
	projectDatabase := ur.cli.Database("mongoDemo")
	projectsCollection := projectDatabase.Collection("projects")
	return projectsCollection
}

func (ur *ProjectRepo) Insert(project domain.Project) (domain.Project, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	projectDto := dao.NewProjectDao(project)

	projectsCollection := ur.getCollection()
	result, err := projectsCollection.InsertOne(ctx, &projectDto)
	if err != nil {
		ur.logger.Println("Error inserting document:", err)
		return domain.Project{}, err
	}

	ur.logger.Printf("Document ID: %v\n", result.InsertedID)
	return project, nil
}

func GetUserById(id string) (domain.User, error) {
	url := fmt.Sprintf("http://user-server:8080/users/id/%s", id)

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

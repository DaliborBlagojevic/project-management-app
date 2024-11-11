package repositories

import (
	"context"
	"fmt"
	"log"
	"os"
	"project-management-app/microservices/users-service/domain"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

type UserRepo struct {
	cli    *mongo.Client
	logger *log.Logger
}

func New(ctx context.Context, logger *log.Logger) (*UserRepo, error) {
	dburi := os.Getenv("MONGO_DB_URI")

	client, err := mongo.NewClient(options.Client().ApplyURI(dburi))
	if err != nil {
		return nil, err
	}

	err = client.Connect(ctx)
	if err != nil {
		return nil, err
	}

	return &UserRepo{
		cli:    client,
		logger: logger,
	}, nil
}

func (pr *UserRepo) Disconnect(ctx context.Context) error {
	err := pr.cli.Disconnect(ctx)
	if err != nil {
		return err
	}
	return nil
}

func (pr *UserRepo) Ping() {
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

func (ur *UserRepo) getCollection() *mongo.Collection {
	userDatabase := ur.cli.Database("mongoDemo")
	usersCollection := userDatabase.Collection("users")
	return usersCollection
}

func (ur *UserRepo) GetAll() (domain.Users, error) {
	// Initialise context (after 5 seconds timeout, abort operation)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	usersCollection := ur.getCollection()

	var users domain.Users
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

func (ur *UserRepo) GetAvailableMembers(projectId string) ([]map[string]interface{}, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	usersCollection := ur.getCollection()
	projectsCollection := ur.cli.Database("mongoDemo").Collection("projects")

	var users domain.Users

	// Konvertujemo projectId u ObjectID
	objID, err := primitive.ObjectIDFromHex(projectId)
	if err != nil {
		ur.logger.Println(err)
		return nil, err
	}

	// Dohvatamo projekat iz baze podataka
	var project struct {
		Members []struct {
			Id primitive.ObjectID `bson:"_id"`
		} `bson:"members"`
	}
	err = projectsCollection.FindOne(ctx, bson.M{"_id": objID}).Decode(&project)
	if err != nil {
		ur.logger.Println(err)
		return nil, err
	}

	// Kreiramo mapu članova projekta za brzu proveru
	memberMap := make(map[primitive.ObjectID]bool)
	for _, member := range project.Members {
		memberMap[member.Id] = true
	}

	// Find all users that are active, with role 3 (member)
	usersCursor, err := usersCollection.Find(ctx, bson.M{
		"isActive": true,
		"role":     3,
	}, options.Find().SetProjection(bson.M{"_id": 1, "username": 1, "name": 1, "surname": 1}))
	if err != nil {
		ur.logger.Println(err)
		return nil, err
	}
	if err = usersCursor.All(ctx, &users); err != nil {
		ur.logger.Println(err)
		return nil, err
	}

	// Filtriraj podatke pre slanja na frontend
	var userResponses []map[string]interface{}
	for _, user := range users {
		// Proveravamo da li korisnik već postoji u listi članova projekta
		if _, exists := memberMap[user.Id]; !exists {
			userResponses = append(userResponses, map[string]interface{}{
				"Id":       user.Id.Hex(),
				"Username": user.Username,
				"Name":     user.Name,
				"Surname":  user.Surname,
			})
		}
	}

	return userResponses, nil
}

func (ur *UserRepo) GetById(id string) (*domain.User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	usersCollection := ur.getCollection()

	var user domain.User
	objID, _ := primitive.ObjectIDFromHex(id)
	err := usersCollection.FindOne(ctx, bson.M{"_id": objID}).Decode(&user)
	if err != nil {
		ur.logger.Println(err)
		return nil, err
	}
	return &user, nil
}

func (ur *UserRepo) GetByUsername(username string) (*domain.User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	usersCollection := ur.getCollection()

	var user domain.User

	err := usersCollection.FindOne(ctx, bson.M{"username": username}).Decode(&user)
	if err != nil {
		ur.logger.Println(err)
		return nil, err
	}
	return &user, nil
}

func (ur *UserRepo) GetByEmail(email string) (domain.Users, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	usersCollection := ur.getCollection()

	var users domain.Users
	usersCursor, err := usersCollection.Find(ctx, bson.M{"username": email})
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

func (ur *UserRepo) GetAllByUsername(username string) (domain.Users, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	usersCollection := ur.getCollection()

	var users domain.Users
	usersCursor, err := usersCollection.Find(ctx, bson.M{"username": username})
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

func (ur *UserRepo) Insert(user domain.User) (domain.User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	usersCollection := ur.getCollection()

	result, err := usersCollection.InsertOne(ctx, &user)
	if err != nil {
		ur.logger.Println(err)
		return domain.User{}, err
	}
	ur.logger.Printf("Documents ID: %v\n", result.InsertedID)
	return user, nil
}

func (pr *UserRepo) ActivateAccount(uuid string, user *domain.User) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	usersCollection := pr.getCollection()

	filter := bson.M{"activationCode": uuid}
	update := bson.M{"$set": bson.M{
		"isActive": true,
	}}
	result, err := usersCollection.UpdateOne(ctx, filter, update)
	pr.logger.Printf("Documents matched: %v\n", result.MatchedCount)
	pr.logger.Printf("Documents updated: %v\n", result.ModifiedCount)

	if err != nil {
		pr.logger.Println(err)
		return err
	}
	return nil
}

func (ur *UserRepo) RemoveExpiredActivationCodes() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	usersCollection := ur.getCollection()

	// Definiši vreme isteka (5 minuta)
	expirationTime := time.Now().Add(-1 * time.Minute)

	// Pronađi sve korisnike čiji je `CreatedAt` stariji od 5 minuta i koji nisu aktivirani
	filter := bson.M{
		"isActive":       false,
		"createdAt":      bson.M{"$lt": expirationTime},
		"activationCode": bson.M{"$ne": nil},
	}

	update := bson.M{
		"$unset": bson.M{"activationCode": ""}, // Briši `activationCode`
	}

	_, err := usersCollection.UpdateMany(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to remove expired activation codes: %v", err)
	}

	return nil
}

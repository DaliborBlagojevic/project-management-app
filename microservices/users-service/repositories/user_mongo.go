package repositories

import (
	"context"
	"fmt"
	"log"
	"os"
	"example.com/project-management-app/microservices/users-service/domain"
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
		"activationCode": "",
	}}

	result, err := usersCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		pr.logger.Println("Error updating user:", err)
		return err
	}

	// Provera da li je pronađen i ažuriran neki dokument
	if result.MatchedCount == 0 {
		errMsg := "Your activation code has expired or is invalid"
		pr.logger.Println(errMsg)
		return domain.ErrCodeExpired()
	}

	pr.logger.Printf("Documents matched: %v\n", result.MatchedCount)
	pr.logger.Printf("Documents updated: %v\n", result.ModifiedCount)

	return nil
}


func (ur *UserRepo) RemoveExpiredActivationCodes() error {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    usersCollection := ur.getCollection()

    // Definiši vreme isteka (2 minuta)
    expirationTime := time.Now().Add(-1 * time.Minute)

    // Pronađi sve korisnike čiji je `CreatedAt` stariji od 2 minuta
    filter := bson.M{
        "createdAt":       bson.M{"$lt": expirationTime},
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


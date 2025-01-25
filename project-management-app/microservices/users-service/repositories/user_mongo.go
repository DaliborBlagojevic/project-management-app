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
	"go.opentelemetry.io/otel/trace"
)

type UserRepo struct {
	cli    *mongo.Client
	logger *log.Logger
	tracer trace.Tracer
}

func New(ctx context.Context, logger *log.Logger, tracer trace.Tracer) (*UserRepo, error) {
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
		tracer: tracer,
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
	userDatabase := ur.cli.Database("users")
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

func (ur *UserRepo) GetAvailableMembers(projectId string, projectMembers []domain.User) ([]domain.User, error) {
	// Fetch project members from the provided list
	projectMembersMap := make(map[string]bool)
	for _, member := range projectMembers {
		projectMembersMap[member.Username] = true
	}

	// Fetch all users
	users, err := ur.GetAll()
	if err != nil {
		ur.logger.Println(err)
		return nil, err
	}

	// Filter users based on role and isActive
	var filteredUsers []domain.User
	for _, user := range users {
		if user.Role == domain.PROJECT_MEMBER && user.IsActive {
			filteredUsers = append(filteredUsers, *user)
		}
	}

	// Filter out users who are already members of the project
	var availableMembers []domain.User
	for _, user := range filteredUsers {
		if !projectMembersMap[user.Username] {
			availableMembers = append(availableMembers, domain.User{
				Username: user.Username,
				Name:     user.Name,
				Surname:  user.Surname,
			})
		}
	}

	// Log available members after filtering
	ur.logger.Println("Available members after filtering:")
	for _, user := range availableMembers {
		ur.logger.Printf("User: %+v\n", user)
	}

	return availableMembers, nil
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

func (ur *UserRepo) GetByUUID(id string) (*domain.User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	usersCollection := ur.getCollection()

	var user domain.User

	err := usersCollection.FindOne(ctx, bson.M{"activationCode": id}).Decode(&user)
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

func (ur *UserRepo) Insert(ctx context.Context ,user domain.User) (domain.User, error) {
    ctx, span := ur.tracer.Start(ctx, "UserService.Insert")
    defer span.End()
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
		"isActive":       true,
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

func (pr *UserRepo) SetRecoveryCode(username string, recoveryCode string, user *domain.User) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	usersCollection := pr.getCollection()

	filter := bson.M{"username": username}
	update := bson.M{"$set": bson.M{
		"recoveryCode": recoveryCode,
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

func (pr *UserRepo) UpdateActivationCode(oldCode string, newCode string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	usersCollection := pr.getCollection()

	// Postavljanje filtera za ažuriranje
	filter := bson.M{"activationCode": oldCode}
	update := bson.M{
		"$set": bson.M{
			"createdAt":      time.Now(),
			"activationCode": newCode,
			"isExpired":      false,
		},
	}

	// Ažuriranje korisnika u bazi
	result, err := usersCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		pr.logger.Println("Error updating user:", err)
		return err
	}

	// Provera da li je neki dokument pronađen i ažuriran
	if result.MatchedCount == 0 {
		errMsg := "Your activation code has expired or is invalid"
		pr.logger.Println(errMsg)
		return domain.ErrCodeExpired()
	}

	pr.logger.Printf("Documents matched: %v\n", result.MatchedCount)
	pr.logger.Printf("Documents updated: %v\n", result.ModifiedCount)

	return nil
}

func (ur *UserRepo) MarkExpiredActivationCodes() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	usersCollection := ur.getCollection()

	// Definiši vreme isteka (2 minuta)
	expirationTime := time.Now().Add(-1 * time.Minute)

	// Pronađi sve korisnike čiji je `CreatedAt` stariji od 2 minuta
	filter := bson.M{
		"createdAt":      bson.M{"$lt": expirationTime},
		"activationCode": bson.M{"$ne": nil},
	}

	// Obeleži aktivacioni kod kao istekao
	update := bson.M{
		"$set": bson.M{"isExpired": true}, // Postavi polje `isExpired` na `true`
	}

	_, err := usersCollection.UpdateMany(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to mark expired activation codes: %v", err)
	}

	return nil
}

func (ur *UserRepo) ChangePassword(ctx context.Context,username string, newPassword string, user *domain.User) error {
    ctx, span := ur.tracer.Start(ctx, "UserRepository.ChangePassword")
    defer span.End()
	usersCollection := ur.getCollection()

	filter := bson.M{"username": username}

	update := bson.M{"$set": bson.M{
		"password": newPassword,
	}}

	result, err := usersCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		ur.logger.Println("Error updating user:", err)
		return err
	}

	ur.logger.Printf("Documents matched: %v\n", result.MatchedCount)
	ur.logger.Printf("Documents updated: %v\n", result.ModifiedCount)

	return nil
}

func (pr *UserRepo) RecoveryPassword(uuid string, password string, user *domain.User) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	usersCollection := pr.getCollection()

	filter := bson.M{"recoveryCode": uuid}
	update := bson.M{"$set": bson.M{
		"password":     password,
		"recoveryCode": "",
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

func (pr *UserRepo) Delete(username string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	patientsCollection := pr.getCollection()

	filter := bson.D{{Key: "username", Value: username}}
	result, err := patientsCollection.DeleteOne(ctx, filter)
	if err != nil {
		pr.logger.Println(err)
		return err
	}
	pr.logger.Printf("Documents deleted: %v\n", result.DeletedCount)
	return nil
}

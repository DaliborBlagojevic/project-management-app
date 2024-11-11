package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	"project-management-app/microservices/proto/user" // Import the generated gRPC proto package

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	// Set up a timeout context
	timeoutContext, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Initialize logger
	logger := log.New(os.Stdout, "[user-client] ", log.LstdFlags)

	// Connect to the gRPC server
	conn, err := grpc.Dial("localhost:8000", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Fatal(err)
	}
	defer conn.Close()

	// Initialize the gRPC client
	userServiceClient := user.NewUserServiceClient(conn)

	// Example: Get all users
	getAllResp, err := userServiceClient.GetUsers(timeoutContext, &user.GetUsersRequest{})
	if err != nil {
		logger.Println("Error while getting all users:", err)
	} else {
		fmt.Println("All Users:", getAllResp.User)
	}

	// Example: Get user by username


	// Set up signal handling for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, os.Kill)

	// Wait for shutdown signal
	sig := <-sigCh
	logger.Println("Received terminate, graceful shutdown", sig)
}

// handleErr is a helper function for error handling
func handleErr(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}

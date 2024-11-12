package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"project-management-app/microservices/users-service/handlers"
	"project-management-app/microservices/users-service/repositories"
	"project-management-app/microservices/users-service/services"

	"github.com/gorilla/mux"
)

func main() {
	// Postavite adresu servera ručno
	address := ":8000" // Zamenite port brojem koji vam odgovara

	// Set up a timeout context
	timeoutContext, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Initialize logger
	storeLogger := log.New(os.Stdout, "[user-store] ", log.LstdFlags)

	// Initialize user repository
	userRepository, err := repositories.New(timeoutContext, storeLogger)
	handleErr(err)

	// Initialize user service
	userService := services.NewUserService(userRepository)

	// Initialize user handler
	userHandler := handlers.NewUserHandler(userService, userRepository)

	// Set up the router
	router := mux.NewRouter()
	router.Use(userHandler.MiddlewareContentTypeSet)

	getRouter := router.Methods(http.MethodGet).Subrouter()
	getRouter.HandleFunc("/users", userHandler.GetAll)
	getRouter.HandleFunc("/users/{username}", userHandler.GetUserByUsername)
	getRouter.HandleFunc("/users/id/{id}", userHandler.GetUserById)
	getRouter.HandleFunc("/projects/{projectId}/availableMembers", userHandler.GetAvailableMembers)

	patchRouter := router.Methods(http.MethodPatch).Subrouter()
	patchRouter.HandleFunc("/users/{uuid}", userHandler.PatchUser)
	patchRouter.Use(userHandler.MiddlewareUserDeserialization)

	router.HandleFunc("/users", userHandler.Create).Methods(http.MethodPost)

	log.Println("Users service is running on", address)
	log.Println("Routes are set up correctly")

	// Set up the server
	server := &http.Server{
		Handler: router,
		Addr:    address,
	}

	// Pokrenite gorutinu za PeriodicCleanup
	go userService.PeriodicCleanup()

	// Pokrenite server
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Could not listen on %s: %v\n", address, err)
		}
	}()

	// Set up signal handling for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, os.Kill)

	// Wait for shutdown signal
	sig := <-sigCh
	log.Println("Received terminate, graceful shutdown", sig)

	// Shutdown the server gracefully
	ctx, cancelShutdown := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelShutdown()
	if err := server.Shutdown(ctx); err != nil {
		log.Fatal("Cannot gracefully shutdown:", err)
	}
	log.Println("Server stopped")
}

// handleErr is a helper function for error handling
func handleErr(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}

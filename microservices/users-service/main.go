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

	gorillaHandlers "github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

func main() {
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
	getRouter.HandleFunc("/users/{username}", userHandler.GetPatientsByName)
	getRouter.HandleFunc("/users/id/{id}", userHandler.GetUserById)

	patchRouter := router.Methods(http.MethodPatch).Subrouter()
	patchRouter.HandleFunc("/auth/{id}", userHandler.PatchUser)
	patchRouter.Use(userHandler.MiddlewareUserDeserialization)

	router.HandleFunc("/users", userHandler.Create).Methods(http.MethodPost)

	cors := gorillaHandlers.CORS(
		gorillaHandlers.AllowedOrigins([]string{"*"}),
		gorillaHandlers.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE", "PATCH"}),
		gorillaHandlers.AllowedHeaders([]string{"Content-Type", "Authorization"}),
	)

	// Set up the server
	port := os.Getenv("PORT")
	if len(port) == 0 {
		port = "8080"
	}
	server := &http.Server{
		Addr:         ":" + port,
		Handler:      cors(router),
		IdleTimeout:  120 * time.Second,
		ReadTimeout:  1 * time.Second,
		WriteTimeout: 1 * time.Second,
	}

	// Start the server in a goroutine
	go func() {
		log.Println("Server listening on port", port)
		if err := server.ListenAndServe(); err != nil {
			log.Fatal(err)
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

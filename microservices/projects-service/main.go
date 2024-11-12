package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"project-management-app/microservices/projects-service/handlers"
	"project-management-app/microservices/projects-service/repositories"
	"project-management-app/microservices/projects-service/services"


	"github.com/gorilla/mux"
)

func main() {
	// Set up a timeout context

	address := ":8000" // Zamenite port brojem koji vam odgovara

	timeoutContext, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Initialize logger
	storeLogger := log.New(os.Stdout, "[projects-store] ", log.LstdFlags)

	projectRepository, err := repositories.New(timeoutContext, storeLogger)
	handleErr(err)

	projectService := services.NewUserService(projectRepository)

	projectHandler := handlers.NewprojectHandler(projectService, projectRepository)

	// Set up the router
	router := mux.NewRouter()
	router.Use(projectHandler.MiddlewareContentTypeSet)

	getRouter := router.Methods(http.MethodGet).Subrouter()
	getRouter.HandleFunc("/projects", projectHandler.GetAll).Methods("GET")

	postRouter := router.Methods(http.MethodPost).Subrouter()
	postRouter.HandleFunc("/projects", projectHandler.Create).Methods("POST")
	postRouter.Use(projectHandler.ProjectContextMiddleware)

	patchRouter := router.Methods(http.MethodPatch).Subrouter()
	patchRouter.HandleFunc("/projects/{id}/addMember", projectHandler.AddMember).Methods("PATCH")


	

	
	server := &http.Server{
		Handler: router,
		Addr:    address,
	}
	log.Fatal(server.ListenAndServe())


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


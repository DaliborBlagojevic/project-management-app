package main

import (
	"context"
	"fmt"
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
	timeoutContext, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	config := loadConfig()

	// Initialize logger
	storeLogger := log.New(os.Stdout, "[user-store] ", log.LstdFlags)

	projectRepository, err := repositories.New(timeoutContext, storeLogger)
	handleErr(err)

	projectService, err := services.NewProjectService(projectRepository)
	handleErr(err)

	projectHandler, err := handlers.NewprojectHandler(projectService)
	handleErr(err)

	// Set up the router
	router := mux.NewRouter()
	router.Use(projectHandler.MiddlewareContentTypeSet)

	getRouter := router.Methods(http.MethodGet).Subrouter()
	postRouter := router.Methods(http.MethodPost).Subrouter()

	getRouter.HandleFunc("/projects", projectHandler.GetAll).Methods("GET")

	postRouter.HandleFunc("/projects", projectHandler.Create).Methods("POST")

	patchRouter := router.Methods(http.MethodPatch).Subrouter()

	// Middleware za deserializaciju korisničkih podataka, primenjen samo na PATCH i POST rute gde je potrebno
	patchRouter.Use(projectHandler.ProjectContextMiddleware)


	

	
	server := &http.Server{
		Handler: router,
		Addr:    config["address"],
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

func loadConfig() map[string]string {
	config := make(map[string]string)
	config["host"] = os.Getenv("HOST")
	config["port"] = os.Getenv("PORT")
	config["address"] = fmt.Sprintf(":%s", os.Getenv("PORT"))
	
	// Adding missing environment variables
	config["db_host"] = os.Getenv("DB_HOST")
	config["db_port"] = os.Getenv("DB_PORT")
	config["db_user"] = os.Getenv("DB_USER")
	config["db_pass"] = os.Getenv("DB_PASS")
	config["db_name"] = os.Getenv("DB_NAME")
	config["mongo_db_uri"] = os.Getenv("MONGO_DB_URI")
	
	return config
}



// handleErr is a helper function for error handling
func handleErr(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}


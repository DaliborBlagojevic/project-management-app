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

	gorillaHandlers "github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

func main() {
	// Set up a timeout context
	timeoutContext, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

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

	// GET subrouter
	// getRouter := router.Methods(http.MethodGet).Subrouter()
	// // Dodajemo GET rute ovde
	// getRouter.HandleFunc("/users", userHandler.GetAllUsers) // Primer rute za dohvat svih korisnika (ako je potrebno)

	// POST subrouter
	getRouter := router.Methods(http.MethodGet).Subrouter()
	postRouter := router.Methods(http.MethodPost).Subrouter()

	// GET ruta za dohvat svih projekata
	getRouter.HandleFunc("/projects", projectHandler.GetAll).Methods("GET")

	// POST ruta za kreiranje novog projekta
	postRouter.HandleFunc("/projects", projectHandler.Create).Methods("POST")

	// PATCH subrouter
	patchRouter := router.Methods(http.MethodPatch).Subrouter()

	// PATCH ruta za dodavanje korisnika u projekat
	patchRouter.HandleFunc("/projects/{id}/addMember", projectHandler.AddMember).Methods("PATCH")

	// Middleware za deserializaciju korisničkih podataka, primenjen samo na PATCH i POST rute gde je potrebno
	//patchRouter.Use(projectHandler.ProjectContextMiddleware)

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

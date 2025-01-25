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
	"project-management-app/microservices/projects-service/config"


	authorizationlib "github.com/Bijelic03/authorizationlibGo"

	"github.com/gorilla/mux"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	
)

func main() {

	address := ":8000"

	cfg := config.GetConfig()

	ctx := context.Background()
	exp, err := newExporter(cfg.JaegerAddress)
	if err != nil {
		log.Fatalf("failed to initialize exporter: %v", err)
	}
	// Create a new tracer provider with a batch span processor and the given exporter.
	tp := newTraceProvider(exp)
	// Handle shutdown properly so nothing leaks.
	defer func() { _ = tp.Shutdown(ctx) }()
	otel.SetTracerProvider(tp)
	// Finally, set the tracer that can be used for this package.
	tracer := tp.Tracer("projects-service")

	otel.SetTextMapPropagator(propagation.TraceContext{})
	// Set up a timeout context
	timeoutContext, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Initialize logger
	storeLogger := log.New(os.Stdout, "[user-store] ", log.LstdFlags)

	taskRepository, err := repositories.NewTaskRepo(timeoutContext, storeLogger, tracer)
	handleErr(err)

	taskService := services.NewTaskService(taskRepository , tracer)

	taskHandler := handlers.NewTaskHandler(taskService, taskRepository, tracer)

	var secretKey = []byte(os.Getenv("SECRET_KEY_AUTH"))

	authHandler := authorizationlib.NewAuthHandler(secretKey)

	// Set up the router
	router := mux.NewRouter()
	router.Use(taskHandler.MiddlewareContentTypeSet)

	privateRouter := router.NewRoute().Subrouter()
	privateRouter.Use(authHandler.MiddlewareAuth)

	managerRouter := router.NewRoute().Subrouter()
	managerRouter.Use(authHandler.MiddlewareAuthManager)

	getRouter := router.Methods(http.MethodGet).Subrouter()

	// Dodajemo GET rute ovde
	getRouter.HandleFunc("/tasks/{id}", taskHandler.GetTasksByProject)
	getRouter.HandleFunc("/tasks", taskHandler.GetAll)
	getRouter.HandleFunc("/tasks/members/{id}", taskHandler.GetMembersByID)
	getRouter.HandleFunc("/tasks/{projectId}/{taskId}/members", taskHandler.FilterMembersNotOnTask)

	// POST subrouter
	//getRouter := router.Methods(http.MethodGet).Subrouter()
	postRouter := managerRouter.Methods(http.MethodPost).Subrouter()

	// GET ruta za dohvat svih projekata
	//getRouter.HandleFunc("/projects", projectHandler.GetAll).Methods("GET")

	// POST ruta za kreiranje novog projekta
	postRouter.HandleFunc("/tasks", taskHandler.Create).Methods("POST") // Route for creating a new task

	// PATCH subrouter
	patchRouter := privateRouter.Methods(http.MethodPatch).Subrouter()
	patchRouter.HandleFunc("/users/{id}", taskHandler.AddMember)
	patchRouter.HandleFunc("/tasks", taskHandler.Update)

	// DELETE subrouter
	deleteRouter := privateRouter.Methods(http.MethodDelete).Subrouter()
	deleteRouter.HandleFunc("/users/{taskId}", taskHandler.RemoveMember)

	// Middleware za deserializaciju korisničkih podataka, primenjen samo na PATCH i POST rute gde je potrebno
	// patchRouter.Use(taskHandler.ProjectContextMiddleware)

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

func newExporter(address string) (*jaeger.Exporter, error) {
	exp, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(address)))
	if err != nil {
		return nil, err
	}
	return exp, nil
}

func newTraceProvider(exp sdktrace.SpanExporter) *sdktrace.TracerProvider {
	// Ensure default SDK resources and the required service name are set.
	r, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String("projects-service"),
		),
	)

	if err != nil {
		panic(err)
	}

	return sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(r),
	)
}

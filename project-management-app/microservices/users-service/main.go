package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"project-management-app/microservices/users-service/config"
	"project-management-app/microservices/users-service/handlers"
	"project-management-app/microservices/users-service/repositories"
	"project-management-app/microservices/users-service/services"

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
	// Postavite adresu servera ručno
	address := ":8000" // Zamenite port brojem koji vam odgovara

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
	tracer := tp.Tracer("users-service")
	otel.SetTextMapPropagator(propagation.TraceContext{})

	// Set up a timeout context
	timeoutContext, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Initialize logger
	storeLogger := log.New(os.Stdout, "[user-store] ", log.LstdFlags)

	// Initialize user repository
	userRepository, err := repositories.New(timeoutContext, storeLogger, tracer)
	handleErr(err)

	// Initialize user service
	userService := services.NewUserService(userRepository, tracer, cfg.ProjectsAddress)
	authService := services.NewAuthService(userRepository, tracer)
	// Initialize user handler
	userHandler := handlers.NewUserHandler(userService, userRepository, tracer)
	authHandler := handlers.NewAuthHandler(authService, tracer)

	var secretKey = []byte(os.Getenv("SECRET_KEY_AUTH"))

	auth := authorizationlib.NewAuthHandler(secretKey)
	// Set up the router
	router := mux.NewRouter()
	router.Use(userHandler.MiddlewareContentTypeSet)
	router.Use(userHandler.ExtractTraceInfoMiddleware)

	privateRouter := router.NewRoute().Subrouter()
	privateRouter.Use(auth.MiddlewareAuth)

	managerRouter := router.NewRoute().Subrouter()
	managerRouter.Use(auth.MiddlewareAuthManager)

	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.Printf("Incoming request: %s %s", r.Method, r.URL.Path)
			next.ServeHTTP(w, r)
		})
	})

	getRouter := router.Methods(http.MethodGet).Subrouter()

	getRouter.HandleFunc("/users", userHandler.GetAll)
	getRouter.HandleFunc("/users/{username}", userHandler.GetUserByUsername)
	getRouter.HandleFunc("/users/id/{id}", userHandler.GetUserById)
	//getRouter.HandleFunc("/projects/{projectId}/availableMembers", userHandler.GetAvailableMembers)

	patchRouter := router.Methods(http.MethodPatch).Subrouter()

	patchRouter.HandleFunc("/users/activate/{uuid}", userHandler.PatchUser)
	patchRouter.HandleFunc("/users/recovery/{uuid}", userHandler.RecoveryPassword)
	patchRouter.HandleFunc("/users/resend/{uuid}", userHandler.ResendActivationCode)
	patchRouter.Use(userHandler.MiddlewareUserDeserialization)

	ChangePasswordPatch := router.Methods(http.MethodPatch).Subrouter()
	ChangePasswordPost := router.Methods(http.MethodPost).Subrouter()
	ChangePasswordPatch.HandleFunc("/users/member/{username}", userHandler.ChangePassword)
	ChangePasswordPost.HandleFunc("/users/recovery", userHandler.SendRecoveryLink)

	postRouter := router.Methods(http.MethodPost).Subrouter()
	postRouter.HandleFunc("/users", userHandler.Create).Methods(http.MethodPost)
	postRouter.HandleFunc("/users/auth", authHandler.LogIn).Methods(http.MethodPost)
	postRouter.HandleFunc("/users/auth/link", userHandler.SendMagicLink).Methods(http.MethodPost)
	postRouter.HandleFunc("/projects/{projectId}/availableMembers", userHandler.GetAvailableMembers).Methods(http.MethodPost)

	deleteRouter := router.Methods(http.MethodDelete).Subrouter()
	deleteRouter.HandleFunc("/users/{username}", userHandler.DeleteUser)

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
			semconv.ServiceNameKey.String("users-service"),
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

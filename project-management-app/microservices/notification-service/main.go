package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"project-management-app/microservices/notification-service/handlers"
	"project-management-app/microservices/notification-service/repositories"
	"project-management-app/microservices/notification-service/services"
	"time"

	"github.com/gorilla/mux"
)

func main() {
	// Postavite adresu servera
	address := ":8000"

	// Set up a timeout context
	timeoutContext, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Initialize logger
	storeLogger := log.New(os.Stdout, "[notifications-store] ", log.LstdFlags)

	// Initialize Cassandra repository
	notificationRepo, err := repositories.NewCassandraRepository(timeoutContext, storeLogger)
	handleErr(err)

	// Initialize notification service
	notificationService := services.NewNotificationService(notificationRepo)

	// Initialize notification handler
	notificationHandler := handlers.NewNotificationHandler(notificationService)

	// Set up the router
	router := mux.NewRouter()

	// Routes for notifications
	getRouter := router.Methods(http.MethodGet).Subrouter()
	getRouter.HandleFunc("/notifications", notificationHandler.GetAllNotificationsByUserID)

	postRouter := router.Methods(http.MethodPost).Subrouter()
	postRouter.HandleFunc("/notifications", notificationHandler.CreateNotification)

	putRouter := router.Methods(http.MethodPut).Subrouter()
	putRouter.HandleFunc("/notifications/read", notificationHandler.MarkAllNotificationsAsRead)

	log.Println("Notifications service is running on", address)
	log.Println("Routes are set up correctly")

	// Set up the server
	server := &http.Server{
		Handler: router,
		Addr:    address,
	}

	// Start the server
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

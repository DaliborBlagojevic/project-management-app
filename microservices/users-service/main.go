package main

import (
	"log"
	"net/http"
	"project-management-app/microservices/users-service/repositories"
	"project-management-app/microservices/users-service/services"
	"project-management-app/microservices/users-service/handlers"
	"github.com/gorilla/mux"
)

func main() {

	userRepository, err := repositories.NewUserInMem()
	handleErr(err)

	userService, err := services.NewUserService(userRepository)
	handleErr(err)
	authService, err := services.NewAuthService(userRepository)
	handleErr(err)

	userHandler, err := handlers.NewUserHandler(userService)
	handleErr(err)
	authHandler, err := handlers.NewAuthHandler(authService)
	handleErr(err)

	authMiddleware, err := handlers.NewAuthMiddleware(authService)
	handleErr(err)



	r := mux.NewRouter()

	// Pod-ruter za rute koje ne zahtevaju autentifikaciju
	publicRoutes := r.PathPrefix("/").Subrouter()
	publicRoutes.HandleFunc("/users", userHandler.Create).Methods("POST")
	publicRoutes.HandleFunc("/auth", authHandler.LogIn).Methods("POST")

	protectedRoutes := r.PathPrefix("/").Subrouter()
	
	// Dodaj autentifikacioni middleware samo na zaštićene rute
	protectedRoutes.Use(authMiddleware.Handle)

	srv := &http.Server{
		Handler: r,
		Addr:    ":8003",
	}
	log.Fatal(srv.ListenAndServe())
}

func handleErr(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}
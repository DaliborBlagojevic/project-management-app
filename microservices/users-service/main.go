package main

import (
	"log"
	"net/http"
	"project-management-app/microservices/users-service/handlers"
	"project-management-app/microservices/users-service/repositories"
	"project-management-app/microservices/users-service/services"

	"github.com/gorilla/mux"
)

func main() {

	userRepository, err := repositories.NewUserInMem()
	handleErr(err)

	userService, err := services.NewUserService(userRepository)
	handleErr(err)


	userHandler, err := handlers.NewUserHandler(userService)
	handleErr(err)




	r := mux.NewRouter()

	// Pod-ruter za rute koje ne zahtevaju autentifikaciju
	publicRoutes := r.PathPrefix("/").Subrouter()
	publicRoutes.HandleFunc("/users", userHandler.Create).Methods("POST")


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
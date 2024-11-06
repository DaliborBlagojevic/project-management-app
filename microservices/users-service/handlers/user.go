package handlers

import (
	"context"
	"log"
	"net/http"
	"net/smtp"
	"project-management-app/microservices/users-service/domain"
	"project-management-app/microservices/users-service/repositories"
	"project-management-app/microservices/users-service/services"

	"github.com/gorilla/mux"
)

type KeyProduct struct{}

type UserHandler struct {
	users *services.UserService
	repo  *repositories.UserRepo
}

func NewUserHandler(s *services.UserService, r *repositories.UserRepo) *UserHandler {
	return &UserHandler{s, r}
}

func (h UserHandler) Create(w http.ResponseWriter, r *http.Request) {
	req := &struct {
		Username string
		Password string
		Name     string
		Surname  string
		Email    string
		Role     string
	}{}
	err := readReq(req, r, w)
	if err != nil {
		return
	}

	user, err := h.users.Create(req.Username, req.Password, req.Name, req.Surname, req.Email, req.Role)
	if err != nil {
		writeErrorResp(err, w)
		return
	}

	resp := struct {
		Id       string
		Username string
		Password string
		Name     string
		Surname  string
		Email    string
		Role     string
		IsActive bool
	}{
		Id:       user.Id.String(),
		Username: user.Username,
		Password: user.Password,
		Name:     user.Name,
		Surname:  user.Surname,
		Email:    user.Email,
		Role:     user.Role.String(),
		IsActive: user.IsActive,
	}
	writeResp(resp, http.StatusCreated, w)

	verifyLink := "http://localhost:5173/login"
	log.Println(verifyLink)
	// Call the email function to send an email after successful user creation
	h.email(req.Email, "", verifyLink)
}

func (p *UserHandler) GetPatientsByName(rw http.ResponseWriter, h *http.Request) {

	vars := mux.Vars(h)
	username := vars["username"]

	users, err := p.repo.GetByUsername(username)
	if err != nil {
		log.Print("Database exception: ", err)
	}

	err = users.ToJSON(rw)
	if err != nil {
		http.Error(rw, "Unable to convert to json", http.StatusInternalServerError)
		log.Fatal("Unable to convert to json :", err)
		return
	}
}

func (p *UserHandler) GetUserById(rw http.ResponseWriter, h *http.Request) {

	vars := mux.Vars(h)
	id := vars["id"]

	user, err := p.repo.GetById(id)
	if err != nil {
		log.Print("Database exception: ", err)
	}

	err = user.ToJSON(rw)
	if err != nil {
		http.Error(rw, "Unable to convert to json", http.StatusInternalServerError)
		log.Fatal("Unable to convert to json :", err)
		return
	}
}

func (p *UserHandler) MiddlewareContentTypeSet(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, h *http.Request) {
		log.Println("Method [", h.Method, "] - Hit path :", h.URL.Path)

		rw.Header().Add("Content-Type", "application/json")

		next.ServeHTTP(rw, h)
	})
}

func (u *UserHandler) PatchUser(rw http.ResponseWriter, h *http.Request) {
	// Log the start of the function
	log.Println("PatchUser handler called")

	vars := mux.Vars(h)
	id := vars["username"]
	log.Println("Extracted ID:", id)

	// Retrieve user from context
	user, ok := h.Context().Value(KeyProduct{}).(*domain.User)
	if !ok {
		log.Println("Failed to retrieve user from context")
		http.Error(rw, "Invalid user data", http.StatusBadRequest)
		return
	}

	log.Println("User retrieved from context:", user)

	// Perform the account activation
	err := u.repo.ActivateAccount(id, user)
	if err != nil {
		log.Println("Error activating account:", err)
		http.Error(rw, "Error activating account", http.StatusInternalServerError)
		return
	}

	log.Println("Account successfully activated for user:", user.Username)
	rw.WriteHeader(http.StatusOK)
}

func (p *UserHandler) MiddlewareUserDeserialization(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, h *http.Request) {
		log.Println("MiddlewareUserDeserialization started")

		user := &domain.User{}
		log.Println(user, "user")
		err := user.FromJSON(h.Body)
		log.Println(err, "error")

		// Check for errors during JSON decoding
		if err != nil {
			http.Error(rw, "Unable to decode JSON", http.StatusBadRequest)
			return
		}

		// Log the deserialized user
		log.Println("Deserialized user:", user)

		// Attach user to context
		ctx := context.WithValue(h.Context(), KeyProduct{}, user)
		h = h.WithContext(ctx)

		log.Println("User added to context. Proceeding to next handler.")

		next.ServeHTTP(rw, h)
	})
}

func (h UserHandler) email(email, body, verifyLink string) {
	from := "dalibor.blagojevic1000@gmail.com"
	pass := "yygc seke csqk yfwo"
	to := email

	// HTML email template with "Verify Your Account" button
	msg := "From: " + from + "\n" +
		"To: " + to + "\n" +
		"Subject: Verify Your Account\n" +
		"MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n" +
		`<!DOCTYPE html>
		<html>
		<body style="font-family: Arial, sans-serif; line-height: 1.6;">
			<h2>Hello,</h2>
			<p>` + body + `</p>
			<p>Please verify your account by clicking the button below:</p>
			<a href="` + verifyLink + `" style="background-color: #4CAF50; color: white; padding: 10px 20px; text-align: center; text-decoration: none; display: inline-block; border-radius: 5px;">
				Verify Your Account
			</a>
			<p>If you did not create an account, you can safely ignore this email.</p>
			<p>Best regards,<br>Project Management App Team</p>
		</body>
		</html>`

	err := smtp.SendMail("smtp.gmail.com:587",
		smtp.PlainAuth("", from, pass, "smtp.gmail.com"),
		from, []string{to}, []byte(msg))

	if err != nil {
		log.Printf("smtp error: %s", err)
		return
	}
	log.Println("Successfully sent email to " + to)
}

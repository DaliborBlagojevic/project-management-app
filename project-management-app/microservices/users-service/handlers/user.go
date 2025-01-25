package handlers

import (
	"context"
	"encoding/json"

	"errors"
	"fmt"
	"log"
	"net/http"
	"net/smtp"
	"project-management-app/microservices/users-service/domain"
	"project-management-app/microservices/users-service/repositories"
	"project-management-app/microservices/users-service/services"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

type KeyProduct struct{}

type UserHandler struct {
	users *services.UserService
	repo  *repositories.UserRepo
	tracer trace.Tracer
}

func NewUserHandler(s *services.UserService, r *repositories.UserRepo, t trace.Tracer) *UserHandler {
	return &UserHandler{s, r, t}
}

func (h UserHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx, span := h.tracer.Start(r.Context(), "UsersHandler.Create")
	defer span.End()

	activationCode := uuid.New().String()

	req := &struct {
		Username       string
		Password       string
		Name           string
		Surname        string
		Email          string
		Role           string
		ActivationCode string
	}{}

	err := readReq(req, r, w)
	if err != nil {
		http.Error(w, "Register error", http.StatusBadRequest)
		return
	}

	user, err := h.users.Create(ctx, req.Username, req.Password, req.Name, req.Surname, req.Email, req.Role, activationCode)
	
	if err != nil {
		if errors.Is(err, domain.ErrUserAlreadyExists()) {
			http.Error(w, "User already exists", http.StatusConflict) // HTTP 409 Conflict
		} else {
			http.Error(w, "Error creating user", http.StatusInternalServerError)
		}
		return
	}

	resp := struct {
		Id             string
		Username       string
		Password       string
		Name           string
		Surname        string
		Email          string
		Role           string
		IsActive       bool
		ActivationCode string
		IsExpired      bool
	}{
		Id:             user.Id.String(),
		Username:       user.Username,
		Password:       user.Password,
		Name:           user.Name,
		Surname:        user.Surname,
		Email:          user.Email,
		Role:           user.Role.String(),
		IsActive:       user.IsActive,
		ActivationCode: user.ActivationCode,
		IsExpired:      user.IsExpired,
	}
	writeResp(resp, http.StatusCreated, w)

	verifyLink := "http://localhost:5173/activate/" + activationCode
	log.Println(verifyLink)

	go h.email(req.Email, "", verifyLink)
}

func (p *UserHandler) GetUserByUsername(rw http.ResponseWriter, h *http.Request) {
	log.Println("pogodio je user service")
	vars := mux.Vars(h)
	username := vars["username"]

	users, err := p.repo.GetByUsername(username)
	if err != nil {
		log.Print("Database exception: ", err)
		return
	}

	err = users.ToJSON(rw)
	if err != nil {
		http.Error(rw, "Unable to convert to json", http.StatusInternalServerError)
		log.Fatal("Unable to convert to json :", err)
		return
	}
}

func (u *UserHandler) GetAll(rw http.ResponseWriter, h *http.Request) {
	ctx, span := u.tracer.Start(h.Context(), "OrderHandler.GetOrder")
	defer span.End()
	log.Println(ctx)
	users, err := u.repo.GetAll()
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

func (h UserHandler) GetAvailableMembers(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	projectId := vars["projectId"]

	var req struct {
		Members []domain.User `json:"members"`
	}

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	users, err := h.users.GetAvailableMembers(projectId, req.Members)
	if err != nil {
		writeErrorResp(err, w)
		return
	}

	// Create a response with only username, name, and surname
	type UserResponse struct {
		Username string `json:"username"`
		Name     string `json:"name"`
		Surname  string `json:"surname"`
	}

	var userResponses []UserResponse
	for _, user := range users {
		userResponses = append(userResponses, UserResponse{
			Username: user.Username,
			Name:     user.Name,
			Surname:  user.Surname,
		})
	}

	writeResp(userResponses, http.StatusOK, w)
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
	id := vars["uuid"]
	log.Println("Extracted ID:", id)

	// Retrieve user from context
	user, ok := h.Context().Value(KeyProduct{}).(*domain.User)
	if !ok {
		log.Println("Failed to retrieve user from context")
		http.Error(rw, "Invalid user data", http.StatusBadRequest)
		return
	}

	log.Println("User retrieved from context:", user)

	// Check if the activation code is expired
	if user.IsExpired {
		log.Println("Activation code is expired for user:", user.Username)
		http.Error(rw, "Activation code has expired", http.StatusConflict) // HTTP 409 Conflict
		return
	}

	// Perform the account activation
	err := u.repo.ActivateAccount(id, user)
	if err != nil {
		if errors.Is(err, domain.ErrCodeExpired()) {
			http.Error(rw, err.Error(), http.StatusConflict) // HTTP 409 Conflict
			return
		}
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

func (h UserHandler) ResendActivationCode(w http.ResponseWriter, r *http.Request) {
	// Dobavljanje starog aktivacionog koda iz URL-a
	vars := mux.Vars(r)
	oldCode := vars["uuid"]

	// Pronađi korisnika po starom aktivacionom kodu
	user, err := h.repo.GetByUUID(oldCode)
	if err != nil || user == nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Proveri da li je korisnik već aktivan
	if user.IsActive {
		http.Error(w, "User is already active", http.StatusBadRequest)
		return
	}

	// Generiši novi aktivacioni kod
	newCode := uuid.New().String()
	user.ActivationCode = newCode
	user.IsExpired = false // Resetovanje polja isExpired

	// Ažuriraj korisnika u bazi
	err = h.repo.UpdateActivationCode(oldCode, newCode)
	if err != nil {
		http.Error(w, "Error updating activation code", http.StatusInternalServerError)
		return
	}

	// Slanje novog aktivacionog email-a
	verifyLink := "http://localhost:5173/activate/" + newCode
	log.Println("New activation link:", verifyLink)

	go h.email(user.Email, "", verifyLink)

	resp := struct {
		Message string `json:"message"`
	}{
		Message: "Activation code resent successfully",
	}
	writeResp(resp, http.StatusOK, w)
}

func (h UserHandler) SendMagicLink(w http.ResponseWriter, r *http.Request) {
	req := &struct {
		Email    string `json:"email"`
		Username string `json:"username"`
	}{}

	err := readReq(&req, r, w)
	if err != nil {
		http.Error(w, "Register error", http.StatusBadRequest)
		return
	}

	user, err := h.repo.GetByUsername(req.Username)
	if err != nil {
		log.Printf("User not found: %v", err)
		writeErrorResp(domain.ErrUserNotFound(), w)
		return
	}

	if user.Email != req.Email {
		writeErrorResp(fmt.Errorf("email does not match for the username"), w)
		return
	}

	token, err := services.CreateToken(*user)
	if err != nil {
		log.Printf("Failed to generate token: %v", err)
		writeErrorResp(err, w)
		return
	}

	magicLink := fmt.Sprintf("http://localhost:5173/magic-login?token=%s&username=%s", token, req.Username)
	body := fmt.Sprintf("Hello %s,\n\nClick the button below to log in to your account:", req.Username)

	h.email(req.Email, body, magicLink)

	from := "dalibor.blagojevic1000@gmail.com"
	pass := "yygc seke csqk yfwo"
	to := req.Email

	// HTML email template with "Verify Your Account" button
	msg := "From: " + from + "\n" +
		"To: " + to + "\n" +
		"Subject: Login to your account\n" +
		"MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n" +
		`<!DOCTYPE html>
		<html>
		<body style="font-family: Arial, sans-serif; line-height: 1.6;">
			<h2>Hello,</h2>
			<p>` + body + `</p>
			<p>Login to your account by clicking the button below:</p>
			<a href="` + magicLink + `" style="background-color: #4CAF50; color: white; padding: 10px 20px; text-align: center; text-decoration: none; display: inline-block; border-radius: 5px;">
				Login
			</a>
			<p>If you did not create an account, you can safely ignore this email.</p>
			<p>Best regards,<br>Project Management App Team</p>
		</body>
		</html>`

	err = smtp.SendMail("smtp.gmail.com:587",
		smtp.PlainAuth("", from, pass, "smtp.gmail.com"),
		from, []string{to}, []byte(msg))

	if err != nil {
		log.Printf("smtp error: %s", err)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message": "Magic link sent successfully"}`))
}

func (u UserHandler) ChangePassword(rw http.ResponseWriter, r *http.Request) {
	ctx, span := u.tracer.Start(r.Context(), "UsersHandler.Create")
	defer span.End()
	log.Println("ChangePassword handler called")

	// Extract the username from the URL path
	vars := mux.Vars(r)
	username := vars["username"]
	log.Printf("Extracted username from URL: %s\n", username)

	// Define a struct to parse the request body
	req := &struct {
		OldPassword string `json:"oldPassword"`
		NewPassword string `json:"newPassword"`
	}{}

	log.Println(req.OldPassword)
	// Parse the request body
	log.Println("Attempting to read the request body...")
	err := readReq(req, r, rw)
	if err != nil {
		http.Error(rw, "Register error", http.StatusBadRequest)
		return
	}

	log.Printf("Parsed request body successfully. OldPassword=%s, NewPassword=%s\n", req.OldPassword, req.NewPassword)

	// Fetch the user by username
	log.Printf("Fetching user by username: %s\n", username)
	user, err := u.repo.GetByUsername(username)
	if err != nil {
		log.Printf("User not found: %v\n", err)
		writeErrorResp(domain.ErrUserNotFound(), rw)
		return
	}

	log.Printf("User fetched successfully: %+v\n", user)

	// Call the service's ChangePassword function
	log.Printf("Calling ChangePassword with username=%s, OldPassword=%s, NewPassword=%s\n", username, req.OldPassword, req.NewPassword)
	err = u.users.ChangePassword(ctx, username, req.OldPassword, req.NewPassword, *user)
	if err != nil {
		if err.Error() == "old password does not match the current password" {
			log.Printf("Old password does not match for user: %s\n", username)
			http.Error(rw, err.Error(), http.StatusBadRequest)
			return
		}
		if errors.Is(err, domain.ErrCodeExpired()) {
			log.Printf("Activation code expired for user: %s\n", username)
			http.Error(rw, err.Error(), http.StatusConflict)
			return
		}
		log.Printf("Error changing password: %v\n", err)
		http.Error(rw, "Error changing password", http.StatusInternalServerError)
		return
	}

	log.Printf("Password changed successfully for user: %s\n", username)
	rw.WriteHeader(http.StatusOK)
}

func (u *UserHandler) RecoveryPassword(rw http.ResponseWriter, h *http.Request) {
	// Log the start of the function
	log.Println("PatchUser handler called")

	vars := mux.Vars(h)
	id := vars["uuid"]
	log.Println("Extracted ID:", id)

	// Retrieve user from context
	user, ok := h.Context().Value(KeyProduct{}).(*domain.User)
	if !ok {
		log.Println("Failed to retrieve user from context")
		http.Error(rw, "Invalid user data", http.StatusBadRequest)
		return
	}

	log.Println("User retrieved from context:", user)

	// Check if the activation code is expired
	if user.IsExpired {
		log.Println("Activation code is expired for user:", user.Username)
		http.Error(rw, "Activation code has expired", http.StatusConflict) // HTTP 409 Conflict
		return
	}

	// Perform the account activation
	err := u.users.RecoveryPassword(id, user.Password, user)
	if err != nil {
		if errors.Is(err, domain.ErrCodeExpired()) {
			http.Error(rw, err.Error(), http.StatusConflict) // HTTP 409 Conflict
			return
		}
		log.Println("Error activating account:", err)
		http.Error(rw, "Error activating account", http.StatusInternalServerError)
		return
	}

	log.Println("Account successfully activated for user:", user.Username)
	rw.WriteHeader(http.StatusOK)
}

func (h *UserHandler) SendRecoveryLink(rw http.ResponseWriter, r *http.Request) {
	recoveryCode := uuid.New().String()

	req := &struct {
		Username string `json: username`
		Email    string `json:"email"`
	}{}

	// Čitanje JSON-a iz tela zahteva
	err := json.NewDecoder(r.Body).Decode(req)
	if err != nil {
		http.Error(rw, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// Proverite da li je email prazan
	if req.Email == "" {
		http.Error(rw, "Email is required", http.StatusBadRequest)
		return
	}

	user, err := h.repo.GetByUsername(req.Username)
	if err != nil {
		log.Printf("User not found: %v", err)
		writeErrorResp(domain.ErrUserNotFound(), rw)
		return
	}

	resetLink := "http://localhost:5173/recovery/" + recoveryCode
	log.Println("Reset link:", resetLink)
	log.Println("Email:", req.Email)

	h.repo.SetRecoveryCode(req.Username, recoveryCode, user)
	go h.emailForPasswordRecovery(req.Email, resetLink)

	rw.WriteHeader(http.StatusOK)
}

func (h UserHandler) emailForPasswordRecovery(email, resetLink string) {
	from := "dalibor.blagojevic1000@gmail.com"
	pass := "yygc seke csqk yfwo"
	to := email

	// HTML email template for password recovery
	msg := "From: " + from + "\n" +
		"To: " + to + "\n" +
		"Subject: Password Recovery\n" +
		"MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n" +
		`<!DOCTYPE html>
		<html>
		<body style="font-family: Arial, sans-serif; line-height: 1.6;">
			<h2>Password Recovery</h2>
			<p>You requested to reset your password. Click the button below to reset it:</p>
			<a href="` + resetLink + `" style="background-color: #007BFF; color: white; padding: 10px 20px; text-align: center; text-decoration: none; display: inline-block; border-radius: 5px;">
				Reset Password
			</a>
			<p>If you did not request a password reset, please ignore this email.</p>
			<p>Best regards,<br>Project Management App Team</p>
		</body>
		</html>`

	err := smtp.SendMail("smtp.gmail.com:587",
		smtp.PlainAuth("", from, pass, "smtp.gmail.com"),
		from, []string{to}, []byte(msg))

	if err != nil {
		log.Printf("SMTP error: %s", err)
		return
	}
	log.Println("Successfully sent password recovery email to " + to)
}

func (u *UserHandler) DeleteUser(rw http.ResponseWriter, h *http.Request) {
	ctx, span := u.tracer.Start(h.Context(), "UserHandler.DeleteUser")
	defer span.End()
	log.Println(ctx)
	// Log the start of the function
	log.Println("DeleteUser handler called")

	// Extract username from URL vars
	vars := mux.Vars(h)
	username := vars["username"]
	log.Println("Extracted username:", username)

	// Retrieve user from repository by username
	user, err := u.repo.GetByUsername(username)
	if err != nil {
		log.Printf("User not found: %v\n", err)
		writeErrorResp(domain.ErrUserNotFound(), rw)
		return
	}

	// Perform the user deletion
	err = u.users.Delete(ctx,user)
	if err != nil {
		if errors.Is(err, domain.ErrCodeExpired()) {
			span.SetStatus(codes.Error, err.Error())
			http.Error(rw, err.Error(), http.StatusConflict) // HTTP 409 Conflict
			return
		}
		log.Println("Error deleting user:", err)
		span.SetStatus(codes.Error, err.Error())
		http.Error(rw, "Error deleting user", http.StatusInternalServerError)
		return
	}

	log.Println("User successfully deleted:", user.Username)
	rw.WriteHeader(http.StatusOK)
}

func (u *UserHandler) ExtractTraceInfoMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

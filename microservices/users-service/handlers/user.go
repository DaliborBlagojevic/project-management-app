package handlers

import (

	"log"
	"net/http"
	"net/smtp"
	"project-management-app/microservices/users-service/repositories"
	"project-management-app/microservices/users-service/services"

)

type KeyProduct struct{}

type UserHandler struct {
	users services.UserService
	repo repositories.UserRepo
}

func NewUserHandler(users services.UserService) (UserHandler, error) {
	return UserHandler{
		users: users,
	}, nil
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
	}{
		Id:       user.Id.String(),
		Username: user.Username,
		Password: user.Password,
		Name:     user.Name,
		Surname:  user.Surname,
		Email:    user.Email,
	}
	writeResp(resp, http.StatusCreated, w)

	// Call the email function to send an email after successful user creation
	h.email(req.Email, "Dragan car", "localhost:8080/users/verify")
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




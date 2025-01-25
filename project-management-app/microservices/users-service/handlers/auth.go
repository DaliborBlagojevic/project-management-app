package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"project-management-app/microservices/users-service/domain"
	"project-management-app/microservices/users-service/services"

	"go.opentelemetry.io/otel/trace"
)

type AuthHandler struct {
	auth *services.AuthService
	tracer trace.Tracer
}

func NewAuthHandler(s *services.AuthService, t trace.Tracer) *AuthHandler {
	return &AuthHandler{s, t}
}

func verifyCaptcha(captchaToken string) (bool, error) {
	data := fmt.Sprintf("secret=%s&response=%s", os.Getenv("CAPTCHA_SECRET"), captchaToken)
	payload := bytes.NewBufferString(data)

	url := "https://www.google.com/recaptcha/api/siteverify"

	resp, err := http.Post(url, "application/x-www-form-urlencoded", payload)
	if err != nil {
		return false, fmt.Errorf("captcha verification request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("captcha verification failed with status: %d", resp.StatusCode)
	}

	var captchaResponse struct {
		Success bool     `json:"success"`
		Errors  []string `json:"error-codes,omitempty"`
	}
	err = json.NewDecoder(resp.Body).Decode(&captchaResponse)
	if err != nil {
		return false, fmt.Errorf("failed to decode captcha response: %w", err)
	}

	if !captchaResponse.Success {
		log.Printf("Captcha verification failed with errors: %v", captchaResponse.Errors)
	}
	return captchaResponse.Success, nil
}




func (h AuthHandler) LogIn(w http.ResponseWriter, r *http.Request) {
	ctx, span := h.tracer.Start(r.Context(), "UsersHandler.LogIn")
	defer span.End()
	req := &struct {
		Username       string
		Password       string
		RecaptchaToken string
	}{}

	err := readReq(req, r, w)
	if err != nil {
		return
	}

	// Verifikacija Captcha tokena
	isValidCaptcha, err := verifyCaptcha(req.RecaptchaToken)
	if err != nil || !isValidCaptcha {
		w.WriteHeader(http.StatusBadRequest)
		w.Header().Set("Content-Type", "application/json")
		if err != nil {
			log.Printf("Error verifying captcha: %v", err)
			w.Write([]byte(fmt.Sprintf(`{"error": "failed to verify captcha: %v"}`, err)))
		} else {
			w.Write([]byte(`{"error": "invalid captcha"}`))
		}
		return
	}

	log.Println("Received login request")

	// Logovanje korisnika
	token, err := h.auth.LogIn(ctx, req.Username, req.Password)
	if err != nil {
		log.Printf("Error in login func %s: %v", req.Username, err)

		// Obrada grešaka vezanih za korisnika
		if err == domain.ErrInvalidCredentials() || err == domain.ErrUserNotFound() {
			w.WriteHeader(http.StatusUnauthorized)
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"error": "incorrect username or password"}`))
			return
		}

		if err == domain.ErrUserNotActive() {
			w.WriteHeader(http.StatusUnauthorized)
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"error": "user not active"}`))
			return
		}

		writeErrorResp(err, w)
		return
	}

	log.Println("Token generated:", token)

	// Odgovor sa generisanim tokenom
	resp := struct {
		Token string `json:"token"`
	}{
		Token: token,
	}
	writeResp(resp, http.StatusOK, w)
}



type AuthMiddleware struct {
	auth services.AuthService
}

func NewAuthMiddleware(auth services.AuthService) (AuthMiddleware, error) {
	return AuthMiddleware{
		auth: auth,
	}, nil
}

func (h AuthHandler) MiddlewareAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Set("Content-Type", "application/json")

		// Extract the Authorization header
		tokenString := r.Header.Get("Authorization")
		if tokenString == "" {
			rw.WriteHeader(http.StatusUnauthorized)
			fmt.Fprint(rw, `{"error": "Missing authorization header"}`)
			return
		}

		// Remove "Bearer " prefix
		tokenString = tokenString[len("Bearer "):]

		// Verify the token and get the user claims
		tokenClaims, err := h.auth.VerifyToken(tokenString)
		if err != nil {
			rw.WriteHeader(http.StatusUnauthorized)
			fmt.Fprint(rw, `{"error": "Invalid token"}`)
			return
		}
		if tokenClaims.Role == domain.Role.String(1) {
			rw.WriteHeader(http.StatusForbidden)
			fmt.Fprint(rw, `{"error": "User is not a member or manager"}`)
			return
		}

		// Add user claims (username and role) to the response headers
		rw.Header().Set("username", tokenClaims.Username)
		rw.Header().Set("role", tokenClaims.Role)

		// Pass the request to the next handler
		next.ServeHTTP(rw, r)
	})
}

func (h AuthHandler) MiddlewareAuthManager(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Set("Content-Type", "application/json")

		// Extract the Authorization header
		tokenString := r.Header.Get("Authorization")
		if tokenString == "" {
			rw.WriteHeader(http.StatusUnauthorized)
			fmt.Fprint(rw, `{"error": "Missing authorization header"}`)
			return
		}

		// Remove "Bearer " prefix
		tokenString = tokenString[len("Bearer "):]

		// Verify the token and get the user claims
		tokenClaims, err := h.auth.VerifyToken(tokenString)
		if err != nil {
			rw.WriteHeader(http.StatusUnauthorized)
			fmt.Fprint(rw, `{"error": "Invalid token"}`)
			return
		}

		if tokenClaims.Role != domain.Role.String(2) {
			rw.WriteHeader(http.StatusForbidden)
			fmt.Fprint(rw, `{"error": "User is not a manager"}`)
			return
		}

		// Add user claims (username and role) to the request headers
		r.Header.Set("username", tokenClaims.Username)
		r.Header.Set("role", tokenClaims.Role)

		// Pass the request to the next handler
		next.ServeHTTP(rw, r)
	})
}

func (h AuthHandler) Auth(w http.ResponseWriter, r *http.Request) {
	log.Println("Auth membeeeeer")

	w.Header().Set("Content-Type", "application/json")
	tokenString := r.Header.Get("Authorization")
	if tokenString == "" {
		http.Error(w, `{"error": "Missing authorization header"}`, http.StatusUnauthorized)
		return
	}

	if len(tokenString) > len("Bearer ") {
		tokenString = tokenString[len("Bearer "):]
	} else {
		http.Error(w, `{"error": "Invalid authorization header format"}`, http.StatusUnauthorized)
		return
	}

	tokenClaims, err := h.auth.VerifyToken(tokenString)
	if err != nil {
		http.Error(w, `{"error": "Invalid token"}`, http.StatusUnauthorized)
		return
	}

	if tokenClaims.Role == domain.Role.String(1) {
		http.Error(w, `{"error": "User is not a manager or member"}`, http.StatusForbidden)
		return
	}

	w.Header().Set("username", tokenClaims.Username)
	w.Header().Set("role", tokenClaims.Role)

	w.WriteHeader(http.StatusOK)

}

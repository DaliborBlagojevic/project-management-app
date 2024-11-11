package handlers

import (
	"fmt"
	"log"
	"net/http"
	"project-management-app/microservices/users-service/domain"
	"project-management-app/microservices/users-service/services"
)

type AuthHandler struct {
	auth *services.AuthService
}

func NewAuthHandler(s *services.AuthService) *AuthHandler {
	return &AuthHandler{s}
}

func (h AuthHandler) LogIn(w http.ResponseWriter, r *http.Request) {
	req := &struct {
		Username string
		Password string
	}{}
	err := readReq(req, r, w)
	if err != nil {
		return
	}

	log.Println("Recived login request")

	token, err := h.auth.LogIn(req.Username, req.Password)
	if err != nil {
		log.Printf("Error in login func %s: %v", req.Username, err)
		if err == domain.ErrInvalidCredentials() {
			w.WriteHeader(http.StatusUnauthorized)
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"error": "incorrect username or password"}`))
			return
		}
		writeErrorResp(err, w)
		return
	}

	log.Println("Token genereted:", token)

	resp := struct {
		Token string
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

// func (m AuthMiddleware) Handle(next http.Handler) http.Handler {
// 	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		token := r.Header.Get("Auth-Token")
// 		log.Println("Received Auth-Token:", token)

// 		authenticated, err := m.auth.ResolveUser(token)
// 		if err != nil {
// 			log.Println("Error in ResolveUser:", err)
// 			http.Error(w, "Unauthorized", http.StatusUnauthorized)
// 			return
// 		}

// 		log.Println("User authenticated:", authenticated.Username)
// 		r = r.WithContext(context.WithValue(r.Context(), "auth", &authenticated))
// 		next.ServeHTTP(w, r)
// 	})
// }

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

		// Add user claims (username and role) to the response headers
		rw.Header().Set("username", tokenClaims.Username)
		rw.Header().Set("role", tokenClaims.Role)

		// Pass the request to the next handler
		next.ServeHTTP(rw, r)
	})
}

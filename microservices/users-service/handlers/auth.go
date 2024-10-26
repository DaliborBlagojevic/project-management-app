package handlers


import (
	"log"
	"context"
	"net/http"
	"project-management-app/microservices/users-service/services"
)

type AuthHandler struct {
	auth services.AuthService
}

func NewAuthHandler(auth services.AuthService) (AuthHandler, error) {
	return AuthHandler{
		auth: auth,
	}, nil
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

	log.Println("Primljen zahtev za login")

	token, err := h.auth.LogIn(req.Username, req.Password)
	if err != nil {
		log.Println("Greška u login funkciji:", err)
		writeErrorResp(err, w)
		return
	}
	log.Println("Token generisan:", token)


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

func (m AuthMiddleware) Handle(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        token := r.Header.Get("Auth-Token")
        log.Println("Received Auth-Token:", token)

        authenticated, err := m.auth.ResolveUser(token)
        if err != nil {
            log.Println("Error in ResolveUser:", err)
            http.Error(w, "Unauthorized", http.StatusUnauthorized)
            return
        }

        log.Println("User authenticated:", authenticated.Username)
        r = r.WithContext(context.WithValue(r.Context(), "auth", &authenticated))
        next.ServeHTTP(w, r)
    })
}



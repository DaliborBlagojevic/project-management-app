package handlers


import (
	"net/http"
	"project-management-app/microservices/users-service/services"
)

type UserHandler struct {
	users services.UserService
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
		Name string
		Surname string
		Email string
		Role  string
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
		Name string
		Surname string
		Email string
	}{
		Id:       user.Id.String(),
		Username: user.Username,
		Password: user.Password,
		Name: user.Name,
		Surname: user.Surname,
		Email: user.Email,
	}
	writeResp(resp, http.StatusCreated, w)
}



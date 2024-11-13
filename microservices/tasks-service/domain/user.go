package domain

import (
	"encoding/json"
	"errors"
	"io"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type User struct {
	Id       primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Username string             `bson:"username" json:"username"`
	Password string             `bson:"password" json:"password"`
	Name     string             `bson:"name" json:"name"`
	Surname  string             `bson:"surname,omitempty" json:"surname"`
	Email    string             `bson:"email" json:"email"`
	Role     Role               `bson:"role" json:"role"`
	IsActive bool               `bson:"isActive" json:"isActive"`
}

type Users []*User

func (p *Users) ToJSON(w io.Writer) error {
	e := json.NewEncoder(w)
	return e.Encode(p)
}

func (p *User) ToJSON(w io.Writer) error {
	e := json.NewEncoder(w)
	return e.Encode(p)
}

func (p *User) FromJSON(r io.Reader) error {
	d := json.NewDecoder(r)
	return d.Decode(p)
}

func (u User) Equals(user User) bool {
	return u.Id == user.Id
}

type UserRepository interface {
	GetById(id string) (*User, error)
	GetByUsername(username string) (Users, error)
	GetAll() (Users, error)
	Insert(user User) (User, error)
	// Update(user User) error
}

type Role int

const (
	UNAUTHORIZED_USER Role = iota + 1
	PROJECT_MANAGER   Role = iota + 2
	PROJECT_MEMBER    Role = iota + 3
)

func (r Role) String() string {
	return [...]string{"UNAUTHORIZED_USER", "PROJECT_MANAGER", "PROJECT_MEMBER"}[r-1]
}
func (r Role) EnumIndex() int {
	return int(r)
}

// Definiši kao funkciju paketa, a ne kao metodu Role
func RoleFromString(s string) (Role, error) {
	switch s {
	case "UNAUTHORIZED_USER":
		return UNAUTHORIZED_USER, nil
	case "PROJECT_MANAGER":
		return PROJECT_MANAGER, nil
	case "PROJECT_MEMBER":
		return PROJECT_MEMBER, nil
	default:
		return 0, errors.New("invalid role")
	}
}

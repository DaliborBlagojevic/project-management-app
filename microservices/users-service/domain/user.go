package domain

import (
	"github.com/google/uuid"
)

type User struct {
	Id       uuid.UUID
	Username string
	Password string
	Name string
	Surname string
	Email string
}

func (u User) Equals(user User) bool {
	return u.Id == user.Id
}

type UserRepository interface {
	Get(id uuid.UUID) (User, error)
	GetAll() ([]User, error)
	Create(user User) (User, error)
	Update(user User) error
}

package domain

import (
	"encoding/json"
	"errors"
	"io"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Task struct {
	Id          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Project     string             `bson:"project" json:"project"`
	Name        string             `bson:"name" json:"name"`
	Description string             `bson:"description" json:"description"`
	Status      Status             `bson:"status" json:"status"`
	Members     Users              `bson:"members,omitempty" json:"members"`
}

type User struct {
	Username string `bson:"username" json:"Username"`
	Name     string `bson:"name" json:"Name"`
	Surname  string `bson:"surname" json:"Surname"`
}

type Users []*User

type Tasks []*Task

func (t *Tasks) ToJSON(w io.Writer) error {
	encoder := json.NewEncoder(w)
	return encoder.Encode(t)
}

func (p *Task) ToJSON(w io.Writer) error {
	e := json.NewEncoder(w)
	return e.Encode(p)
}

func (p *Task) FromJSON(r io.Reader) error {
	d := json.NewDecoder(r)
	return d.Decode(p)
}

func (p *Users) ToJSON(w io.Writer) error {
	e := json.NewEncoder(w)
	return e.Encode(p)
}

func (p *Users) FromJSON(r io.Reader) error {
	d := json.NewDecoder(r)
	return d.Decode(p)
}

func (p *User) ToJSON(w io.Writer) error {
	e := json.NewEncoder(w)
	return e.Encode(p)
}

func (p *User) FromJSON(r io.Reader) error {
	d := json.NewDecoder(r)
	return d.Decode(p)
}

func (u Task) MarshalJSON() ([]byte, error) {
	type Alias Task

	return json.Marshal(&struct {
		Status string `json:"status"`
		*Alias
	}{
		Status: u.Status.String(),
		Alias:  (*Alias)(&u),
	})
}

type Status int

const (
	PENDING Status = iota + 1
	IN_PROGRESS
	FINISHED
)

func (r Status) String() string {
	return [...]string{"PENDING", "IN_PROGRESS", "FINISHED"}[r-1]
}
func (r Status) EnumIndex() int {
	return int(r)
}

func StatusFromString(s string) (Status, error) {
	switch s {
	case "PENDING":
		return PENDING, nil
	case "IN_PROGRESS":
		return IN_PROGRESS, nil
	case "FINISHED":
		return FINISHED, nil
	default:
		return 0, errors.New("invalid status")
	}
}

package domain

import (
	"encoding/json"
	"io"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type User struct {
	Id       primitive.ObjectID `bson:"_id,omitempty" json:"Id"`
	Username string             `bson:"username" json:"Username"`
	Name     string             `bson:"name" json:"Name"`
	Surname  string             `bson:"surname" json:"Surname"`
}

type Users []*User

type Project struct {
	Id         primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Manager    User               `bson:"manager" json:"manager"`
	Name       string             `bson:"name" json:"name"`
	EndDate    time.Time          `bson:"end_date" json:"end_date"`
	MinWorkers int                `bson:"min_workers,omitempty" json:"min_workers"`
	MaxWorkers int                `bson:"max_workers" json:"max_workers"`
	Members    Users              `bson:"members,omitempty" json:"members"`
}

type Projects []*Project

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

func (p *Projects) ToJSON(w io.Writer) error {
	e := json.NewEncoder(w)
	return e.Encode(p)
}

func (p *Project) ToJSON(w io.Writer) error {
	e := json.NewEncoder(w)
	return e.Encode(p)
}

func (p *Project) FromJSON(r io.Reader) error {
	d := json.NewDecoder(r)
	return d.Decode(p)
}

func (u *Project) Equals(other *Project) bool { // promena parametra u pokazivač
	return u.Id == other.Id
}

type ProjectRepository interface {
	GetById(id string) (*Project, error)
	GetAllByManager(managerID string) (Projects, error) // promenjeno ime parametra u managerID
	GetAll() (Projects, error)
	Insert(project Project) (Project, error)
	Update(project Project) error
}

package domain

import (

	"encoding/json"
	"io"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Task struct {
	Id       primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Project    primitive.ObjectID `bson:"project" json:"project"`
	Name       string             `bson:"name" json:"name"`
	Description    string             `bson:"description" json:"description"`
	Status Status             `bson:"status" json:"status"`
}


type Tasks []*Task

func (t *Tasks) ToJSON(w io.Writer) error {
	encoder := json.NewEncoder(w)
	return encoder.Encode(t)
}


type Status int

const (
	PENDING Status = iota + 1
	FINISHED   Status = iota + 2
	
)

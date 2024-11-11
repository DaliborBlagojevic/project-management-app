package dao

import (
	"project-management-app/microservices/projects-service/domain"
	"strconv"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ProjectDao struct {
	Manager    primitive.ObjectID `json:"manager"`
	Name       string             `json:"name"`
	EndDate    string             `json:"end_date"`
	MinWorkers string             `json:"min_workers"`
	MaxWorkers string             `json:"max_workers"`
	Members    domain.Users       `json:"members"`
}

func (dao *ProjectDao) ToDomain() (domain.Project, error) {
	endDate, err := time.Parse("2006-01-02", dao.EndDate)
	if err != nil {
		return domain.Project{}, err
	}

	minWorkers, err := strconv.Atoi(dao.MinWorkers)
	if err != nil {
		return domain.Project{}, err
	}

	maxWorkers, err := strconv.Atoi(dao.MaxWorkers)
	if err != nil {
		return domain.Project{}, err
	}

	return domain.Project{
		Manager:    domain.User{Id: dao.Manager},
		Name:       dao.Name,
		EndDate:    endDate,
		MinWorkers: minWorkers,
		MaxWorkers: maxWorkers,
		Members:    domain.Users{},
	}, nil
}

func NewProjectDao(project domain.Project) ProjectDao {
	return ProjectDao{
		Manager:    project.Manager.Id,
		Name:       project.Name,
		EndDate:    project.EndDate.Format("2006-01-02"),
		MinWorkers: strconv.Itoa(project.MinWorkers),
		MaxWorkers: strconv.Itoa(project.MaxWorkers),
		Members:    project.Members,
	}
}

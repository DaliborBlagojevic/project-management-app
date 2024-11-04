package dto

import (
	"project-management-app/microservices/projects-service/domain"
	"strconv"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ProjectDto struct {
	Id         primitive.ObjectID `json:"id"`
	Manager    primitive.ObjectID `json:"manager"`
	Name       string             `json:"name"`
	EndDate    string             `json:"end_date"`
	MinWorkers string             `json:"min_workers"`
	MaxWorkers string             `json:"max_workers"`
}

func (dto *ProjectDto) ToDomain() (domain.Project, error) {
	endDate, err := time.Parse("2006-01-02", dto.EndDate)
	if err != nil {
		return domain.Project{}, err
	}

	minWorkers, err := strconv.Atoi(dto.MinWorkers)
	if err != nil {
		return domain.Project{}, err
	}

	maxWorkers, err := strconv.Atoi(dto.MaxWorkers)
	if err != nil {
		return domain.Project{}, err
	}

	return domain.Project{
		Id:         dto.Id,
		Manager:    domain.User{Id: dto.Manager},
		Name:       dto.Name,
		EndDate:    endDate,
		MinWorkers: minWorkers,
		MaxWorkers: maxWorkers,
	}, nil
}

func NewProjectDto(project domain.Project) ProjectDto {
	return ProjectDto{
		Id:         project.Id,
		Manager:    project.Manager.Id,
		Name:       project.Name,
		EndDate:    project.EndDate.Format("2006-01-02"),
		MinWorkers: strconv.Itoa(project.MinWorkers),
		MaxWorkers: strconv.Itoa(project.MaxWorkers),
	}
}

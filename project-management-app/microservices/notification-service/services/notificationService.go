package services

import (
	"project-management-app/microservices/notification-service/domain"
	"project-management-app/microservices/notification-service/repositories"
)

type NotificationService struct {
	repo *repositories.CassandraRepository
}

func NewNotificationService(repo *repositories.CassandraRepository) *NotificationService {
	return &NotificationService{repo: repo}
}

func (s *NotificationService) GetAllNotificationsByUserID(userID string) ([]domain.Notification, error) {
	return s.repo.GetAllNotificationsByUserID(userID)
}

func (s *NotificationService) CreateNotification(notification *domain.Notification) error {
	return s.repo.CreateNotification(notification)
}

func (s *NotificationService) MarkAllNotificationsAsRead(userID string) error {
    return s.repo.MarkAllNotificationsAsRead(userID)
}

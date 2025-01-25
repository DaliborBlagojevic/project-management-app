package handlers

import (
	"encoding/json"
	"net/http"
	"project-management-app/microservices/notification-service/domain"
	"project-management-app/microservices/notification-service/services"
	"time"
)

type NotificationHandler struct {
	service *services.NotificationService
}

func NewNotificationHandler(service *services.NotificationService) *NotificationHandler {
	return &NotificationHandler{service: service}
}

func (h *NotificationHandler) GetAllNotificationsByUserID(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		http.Error(w, "Missing user_id", http.StatusBadRequest)
		return
	}

	notifications, err := h.service.GetAllNotificationsByUserID(userID)
	if err != nil {
		http.Error(w, "Failed to fetch notifications", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(notifications)
}

func (h *NotificationHandler) CreateNotification(w http.ResponseWriter, r *http.Request) {
	var notification domain.Notification

	// Decode JSON iz tela zahteva
	if err := json.NewDecoder(r.Body).Decode(&notification); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	// Postavljanje trenutnog vremena za CreatedAt
	notification.CreatedAt = time.Now()

	// Kreiranje notifikacije preko servisa
	if err := h.service.CreateNotification(&notification); err != nil {
		http.Error(w, "Failed to create notification", http.StatusInternalServerError)
		return
	}

	// Vraćanje statusa 201 Created
	w.WriteHeader(http.StatusCreated)
}


func (h *NotificationHandler) MarkAllNotificationsAsRead(w http.ResponseWriter, r *http.Request) {
	// Parse userID from query parameters
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		http.Error(w, "user_id query parameter is required", http.StatusBadRequest)
		return
	}

	// Call the service to mark notifications as read
	err := h.service.MarkAllNotificationsAsRead(userID)
	
	if err != nil {
		http.Error(w, "Failed to mark notifications as read: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Respond with success
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "All notifications marked as read"})
}


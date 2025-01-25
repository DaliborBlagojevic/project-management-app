package repositories

import (
	"context"
	"fmt"
	"log"
	"project-management-app/microservices/notification-service/domain"
	"strings"
	"time"

	"github.com/gocql/gocql"
)

type CassandraRepository struct {
	session *gocql.Session
	logger  *log.Logger
}

func NewCassandraRepository(ctx context.Context, logger *log.Logger) (*CassandraRepository, error) {
	cluster := gocql.NewCluster("cassandra-db")
	cluster.Keyspace = "system" // Počnite sa 'system' keyspace za globalne upite
	cluster.Consistency = gocql.Quorum

	logger.Println("Attempting to connect to Cassandra...")

	session, err := cluster.CreateSession()
	if err != nil {
		logger.Printf("Failed to connect to Cassandra: %v\n", err)
		return nil, err
	}
	defer session.Close()

	// Proverite i kreirajte keyspace ako ne postoji
	if err := EnsureKeyspaceExists(session); err != nil {
		logger.Printf("Failed to ensure keyspace exists: %v\n", err)
		return nil, err
	}

	// Povežite se na 'notifications' keyspace
	cluster.Keyspace = "notifications"
	session, err = cluster.CreateSession()
	if err != nil {
		logger.Printf("Failed to connect to notifications keyspace: %v\n", err)
		return nil, err
	}

	// Proveri i kreiraj tabelu ako ne postoji
	if err := EnsureTableExists(session, logger); err != nil {
		logger.Printf("Failed to ensure table exists: %v\n", err)
		session.Close()
		return nil, err
	}

	logger.Println("Connected to Cassandra with keyspace notifications")
	return &CassandraRepository{session: session, logger: logger}, nil
}



func (r *CassandraRepository) Close() {
	r.session.Close()
}

func EnsureKeyspaceExists(session *gocql.Session) error {
	query := `
	CREATE KEYSPACE IF NOT EXISTS notifications
	WITH replication = {
		'class': 'SimpleStrategy',
		'replication_factor': 1
	};
	`
	return session.Query(query).Exec()
}

func EnsureTableExists(session *gocql.Session, logger *log.Logger) error {
	query := `
	CREATE TABLE IF NOT EXISTS notifications (
		id UUID,
		user_id TEXT,
		message TEXT,
		created_at TIMESTAMP,
		is_read BOOLEAN,
		PRIMARY KEY (user_id, created_at)
	) WITH CLUSTERING ORDER BY (created_at DESC);`

	logger.Println("Ensuring notifications table exists...")
	if err := session.Query(query).Exec(); err != nil {
		logger.Printf("Failed to ensure table exists: %v\n", err)
		return err
	}

	logger.Println("Notifications table is ready.")
	return nil
}


func (r *CassandraRepository) GetAllNotificationsByUserID(userID string) ([]domain.Notification, error) {
	var notifications []domain.Notification

	iter := r.session.Query("SELECT id, user_id, message, created_at, is_read FROM notifications WHERE user_id = ?", userID).Iter()
	var notification domain.Notification

	for iter.Scan(&notification.ID, &notification.UserID, &notification.Message, &notification.CreatedAt, &notification.IsRead) {
		notifications = append(notifications, notification)
	}

	if err := iter.Close(); err != nil {
		return nil, err
	}

	return notifications, nil
}

func (r *CassandraRepository) CreateNotification(notification *domain.Notification) error {
	return r.session.Query(
		"INSERT INTO notifications (id, user_id, message, created_at, is_read) VALUES (?, ?, ?, ?, ?)",
		gocql.TimeUUID(), notification.UserID, notification.Message, notification.CreatedAt, notification.IsRead,
	).Exec()
}


func (r *CassandraRepository) MarkNotificationAsRead(userID string, createdAt time.Time) error {
    query := `
        UPDATE notifications
        SET is_read = true
        WHERE user_id = ? AND created_at = ?;`

    if err := r.session.Query(query, userID, createdAt).Exec(); err != nil {
        fmt.Printf("Error updating notification for user_id=%s, created_at=%s: %v\n", userID, createdAt, err)
        return err
    }

    fmt.Printf("Successfully updated notification for user_id=%s, created_at=%s\n", userID, createdAt.Format(time.RFC3339))
    return nil
}



func (r *CassandraRepository) MarkAllNotificationsAsRead(userID string) error {
    query := `SELECT created_at FROM notifications WHERE user_id = ?;`
    fmt.Printf("Executing SELECT query for user_id: %s\n", userID)

    iter := r.session.Query(query, userID).Iter()
	var createdAt time.Time


	fmt.Printf("Fetching notifications for user_id: '%s'\n", userID)
	userID = strings.TrimSpace(userID)
	fmt.Printf("Cleaned user_id: '%s'\n", userID)
	

	if iter.NumRows() == 0 {
		fmt.Printf("No rows found for user_id: %s\n", userID)
	}

	for iter.Scan(&createdAt) {
		fmt.Printf("Fetched created_at: %s for user_id: %s\n", createdAt.Format(time.RFC3339), userID)
	
		if err := r.MarkNotificationAsRead(userID, createdAt); err != nil {
			fmt.Printf("Error marking notification as read for created_at: %s, error: %v\n", createdAt, err)
			return err
		}
	}
	

    if err := iter.Close(); err != nil {
        fmt.Printf("Error closing iterator for user_id: %s, error: %v\n", userID, err)
        return err
    }


    fmt.Printf("Successfully marked all notifications as read for user_id: %s\n", userID)
	fmt.Printf("Attempting to update notification for user_id=%s, created_at=%s\n", userID, createdAt)
    return nil
}




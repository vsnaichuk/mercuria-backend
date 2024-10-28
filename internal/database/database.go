package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"maps"
	"os"
	"slices"
	"strconv"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	_ "github.com/joho/godotenv/autoload"
)

type InviteStatus string

const (
	InvitePending  InviteStatus = "Pending"
	InviteAccepted InviteStatus = "Accepted"
	InviteDeclined InviteStatus = "Declined"
)

type Invite struct {
	ID        int          `json:"id"`
	UserID    int          `json:"user_id"`
	EventID   int          `json:"event_id"`
	Status    InviteStatus `json:"status"`
	CreatedAt time.Time    `json:"created_at"`
	ExpiresAt time.Time    `json:"expires_at"`
}

type Event struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	OwnerID   string    `json:"owner_id"`
	ImageURL  string    `json:"image_url"`
	Owner     User      `json:"owner"`
	Likes     []Like    `json:"likes"`
	Members   []User    `json:"members"`
}

type Like struct {
	ID        int    `json:"id"`
	UserID    string `json:"user_id"`
	EventID   string `json:"event_id"`
	CreatedAt string `json:"created_at"`
}

type User struct {
	ID        string `json:"id"`
	OAuthId   string `json:"oauth_id"`
	Name      string `json:"name"`
	AvatarUrl string `json:"avatar_url"`
	Email     string `json:"email"`
}

// Service represents a service that interacts with a database.
type Service interface {
	AddEventMember(userId string, eventId string) string
	LikeEvent(userId string, eventId string) string
	DislikeEvent(userId string, eventId string) string
	GetUserEvents(userId string) []*Event
	GetEvent(eventId string) (*Event, error)
	CreateEvent(einfo Event) string
	GetOrCreateUser(uinfo User) map[string]string
	Health() map[string]string
	Close() error
}

type service struct {
	db *sql.DB
}

var (
	database   = os.Getenv("DB_DATABASE")
	password   = os.Getenv("DB_PASSWORD")
	username   = os.Getenv("DB_USERNAME")
	port       = os.Getenv("DB_PORT")
	host       = os.Getenv("DB_HOST")
	schema     = os.Getenv("DB_SCHEMA")
	url        = os.Getenv("DB_URL")
	dbInstance *service
)

func New() Service {
	// Reuse Connection
	if dbInstance != nil {
		return dbInstance
	}
	db, err := sql.Open("pgx", url)
	if err != nil {
		log.Fatal(err)
	}
	dbInstance = &service{
		db: db,
	}
	return dbInstance
}

func (s *service) AddEventMember(userId string, eventId string) string {
	_, err := s.db.Query("INSERT INTO members (user_id, event_id) VALUES ($1, $2)", userId, eventId)
	if err != nil {
		log.Fatalf("AddEventMemberQuery %v", err)
	}
	return "success"
}

// func (s *service) CreateEventInvite(eventId string, createdBy string) string {
// 	var id string
// 	err := s.db.QueryRow("INSERT INTO invites (event_id, created_by) VALUES ($1, $2) RETURNING id",
// 		eventId,
// 		createdBy,
// 	).Scan(&id)

// 	if err != nil {
// 		log.Fatalf("CreateEventInviteQuery %v", err)
// 	}
// 	return id
// }

func (s *service) LikeEvent(userId string, eventId string) string {
	_, err := s.db.Query("INSERT INTO likes (user_id, event_id) VALUES ($1, $2)", userId, eventId)
	if err != nil {
		log.Fatalf("LikeEventQuery %v", err)
	}
	return "success"
}

func (s *service) DislikeEvent(userId string, eventId string) string {
	_, err := s.db.Query("DELETE FROM likes WHERE user_id = $1 AND event_id = $2;", userId, eventId)
	if err != nil {
		log.Fatalf("DislikeEventQuery %v", err)
	}
	return "success"
}

func (s *service) GetUserEvents(userId string) []*Event {
	rows, err := s.db.Query("SELECT * FROM public.get_events($1)", userId)
	if err != nil {
		log.Fatalf("GetUserEvents %v", err)
	}
	defer rows.Close()

	events, err := ScanEventRows(rows)
	if err != nil {
		log.Fatalf("GetUserEventsScan %v", err)
	}

	slice := slices.Collect(maps.Values(events))
	slices.SortFunc(slice, func(a, b *Event) int {
		return b.CreatedAt.Compare(a.CreatedAt)
	})
	return slice
}

func (s *service) GetEvent(eventId string) (*Event, error) {
	rows, err := s.db.Query("SELECT * FROM public.get_event($1)", eventId)
	if err != nil {
		log.Fatalf("GetEventQuery %v", err)
	}
	defer rows.Close()

	events, err := ScanEventRows(rows)
	if err != nil {
		log.Fatalf("GetEventScan %v", err)
	}

	if event, exists := events[eventId]; exists {
		return event, nil
	}
	return nil, fmt.Errorf("event with ID %s not found", eventId)
}

func (s *service) CreateEvent(einfo Event) string {
	var id string
	err := s.db.QueryRow("SELECT * FROM public.create_event($1, $2)",
		einfo.Name,
		einfo.OwnerID,
	).Scan(&id)

	if err != nil {
		log.Fatalf("CreateEventQuery  %v", err)
	}
	return id
}

func (s *service) GetOrCreateUser(uinfo User) map[string]string {
	var id, name, avatarUrl string
	err := s.db.QueryRow(
		"SELECT * FROM public.get_or_create_user($1, $2, $3, $4)",
		uinfo.OAuthId,
		uinfo.Name,
		uinfo.AvatarUrl,
		uinfo.Email,
	).Scan(&id, &name, &avatarUrl)

	if err != nil {
		log.Fatalf("GetOrCreateUserQuery  %v", err)
	}

	data := make(map[string]string)
	data["id"] = id
	data["name"] = name
	data["avatar_url"] = avatarUrl
	return data
}

func (s *service) Health() map[string]string {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	stats := make(map[string]string)

	// Ping the database
	err := s.db.PingContext(ctx)
	if err != nil {
		stats["status"] = "down"
		stats["error"] = fmt.Sprintf("db down: %v", err)
		log.Fatalf(fmt.Sprintf("db down: %v", err)) // Log the error and terminate the program
		return stats
	}

	stats["status"] = "up"
	stats["message"] = "It's healthy"

	// Get database stats (like open connections, in use, idle, etc.)
	dbStats := s.db.Stats()
	stats["open_connections"] = strconv.Itoa(dbStats.OpenConnections)
	stats["in_use"] = strconv.Itoa(dbStats.InUse)
	stats["idle"] = strconv.Itoa(dbStats.Idle)
	stats["wait_count"] = strconv.FormatInt(dbStats.WaitCount, 10)
	stats["wait_duration"] = dbStats.WaitDuration.String()
	stats["max_idle_closed"] = strconv.FormatInt(dbStats.MaxIdleClosed, 10)
	stats["max_lifetime_closed"] = strconv.FormatInt(dbStats.MaxLifetimeClosed, 10)

	if dbStats.OpenConnections > 40 {
		stats["message"] = "The database is experiencing heavy load."
	}
	if dbStats.WaitCount > 1000 {
		stats["message"] = "The database has a high number of wait events, indicating potential bottlenecks."
	}
	if dbStats.MaxIdleClosed > int64(dbStats.OpenConnections)/2 {
		stats["message"] = "Many idle connections are being closed, consider revising the connection pool settings."
	}
	if dbStats.MaxLifetimeClosed > int64(dbStats.OpenConnections)/2 {
		stats["message"] = "Many connections are being closed due to max lifetime, consider increasing max lifetime or revising the connection usage pattern."
	}

	return stats
}

func (s *service) Close() error {
	log.Printf("Disconnected from database: %s", database)
	return s.db.Close()
}

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

type Event struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	Owner     string    `json:"owner"`
	ImageURL  string    `json:"image_url"`
	OwnerData User      `json:"owner_data"`
	Likes     []Like    `json:"likes"`
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
	LikeEvent(uid string, eid string) string
	DislikeEvent(uid string, eid string) string
	GetEvents() []*Event
	GetEvent(eid string) (*Event, error)
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
	dbInstance *service
)

func New() Service {
	// Reuse Connection
	if dbInstance != nil {
		return dbInstance
	}
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable&search_path=%s", username, password, host, port, database, schema)
	db, err := sql.Open("pgx", connStr)
	if err != nil {
		log.Fatal(err)
	}
	dbInstance = &service{
		db: db,
	}
	return dbInstance
}

func (s *service) LikeEvent(uid string, eid string) string {
	_, err := s.db.Query("INSERT INTO likes (user_id, event_id) VALUES ($1, $2)", uid, eid)
	if err != nil {
		log.Fatalf("LikeEventQuery %v", err)
	}
	return "success"
}

func (s *service) DislikeEvent(uid string, eid string) string {
	_, err := s.db.Query("DELETE FROM likes WHERE user_id = $1 AND event_id = $2;", uid, eid)
	if err != nil {
		log.Fatalf("DislikeEventQuery %v", err)
	}
	return "success"
}

func (s *service) GetEvents() []*Event {
	rows, err := s.db.Query("SELECT * FROM public.get_events()")
	if err != nil {
		log.Fatalf("GetEvents %v", err)
	}
	defer rows.Close()

	events, err := s.scanEventRows(rows)
	if err != nil {
		log.Fatalf("GetEventsScan %v", err)
	}

	slice := slices.Collect(maps.Values(events))
	slices.SortFunc(slice, func(a, b *Event) int { 
		return b.CreatedAt.Compare(a.CreatedAt) 
	})
	return slice
}

func (s *service) GetEvent(eid string) (*Event, error) {
	rows, err := s.db.Query("SELECT * FROM public.get_event($1)", eid)
	if err != nil {
		log.Fatalf("GetEventQuery %v", err)
	}
	defer rows.Close()

	events, err := s.scanEventRows(rows)
	if err != nil {
		log.Fatalf("GetEventScan %v", err)
	}

	if event, exists := events[eid]; exists {
		return event, nil
	}
	return nil, fmt.Errorf("event with ID %s not found", eid)
}

func (s *service) CreateEvent(einfo Event) string {
	var id string
	err := s.db.QueryRow("SELECT * FROM public.create_event($1, $2)",
		einfo.Name,
		einfo.Owner,
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

// --- Helpers

func (s *service) scanEventRows(rows *sql.Rows) (map[string]*Event, error) {
	events := make(map[string]*Event)

	for rows.Next() {
		var id, name, owner, image string
		var created time.Time
		var ownerID, ownerAuthID, ownerName, ownerAvatar, ownerEmail string
		var likeID sql.NullInt32
		var likeUID, likeEventID, likeCreated sql.NullString

		// Scan the row into appropriate variables
		err := rows.Scan(&id, &name, &created, &owner, &image,
			&ownerID, &ownerAuthID, &ownerName, &ownerAvatar, &ownerEmail,
			&likeID, &likeUID, &likeEventID, &likeCreated)

		if err != nil {
			return nil, fmt.Errorf("processEventRows: %v", err)
		}

		if _, exists := events[id]; !exists {
			events[id] = &Event{
				ID:        id,
				Name:      name,
				CreatedAt: created,
				Owner:     owner,
				ImageURL:  image,
				OwnerData: User{
					ID:        ownerID,
					OAuthId:   ownerAuthID,
					Name:      ownerName,
					AvatarUrl: ownerAvatar,
					Email:     ownerEmail,
				},
				Likes: []Like{},
			}
		}

		if likeID.Valid {
			events[id].Likes = append(events[id].Likes, Like{
				ID:        int(likeID.Int32),
				UserID:    likeUID.String,
				EventID:   likeEventID.String,
				CreatedAt: likeCreated.String,
			})
		}
	}

	return events, nil
}

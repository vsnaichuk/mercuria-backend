package database

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	_ "github.com/joho/godotenv/autoload"
)

func ScanEventRows(rows *sql.Rows) (map[string]*Event, error) {
	events := make(map[string]*Event)
	likes := make(map[int]*Like)

	for rows.Next() {
		event, like, member, err := scanNextEvent(rows)
		if err != nil {
			return nil, fmt.Errorf("scanEventRows: %w", err)
		}
		if _, exists := events[event.ID]; !exists {
			events[event.ID] = event
		}
		if like != nil {
			if _, exists := likes[like.ID]; !exists {
				likes[like.ID] = like
				events[event.ID].Likes = append(events[event.ID].Likes, *like)
			}
		}
		events[event.ID].Members = append(events[event.ID].Members, *member)
	}

	return events, nil
}

func scanNextEvent(rows *sql.Rows) (*Event, *Like, *User, error) {
	var id, name, owner, image string
	var created time.Time
	var ownerID, ownerAuthID, ownerName string
	var ownerAvatar, ownerEmail string
	var likeID sql.NullInt32
	var likeUID, likeEID, likeCreated sql.NullString
	var mbrID, mbrAuthID, mbrName string
	var mbrAvatar, mbrEmail string

	if err := rows.Scan(&id, &name, &created, &owner, &image,
		&ownerID, &ownerAuthID, &ownerName, &ownerAvatar, &ownerEmail,
		&likeID, &likeUID, &likeEID, &likeCreated,
		&mbrID, &mbrAuthID, &mbrName, &mbrAvatar, &mbrEmail); err != nil {
		return nil, nil, nil, err
	}

	event := &Event{
		ID:        id,
		Name:      name,
		CreatedAt: created,
		OwnerID:   ownerID,
		ImageURL:  image,
		Owner: User{
			ID:        ownerID,
			OAuthId:   ownerAuthID,
			Name:      ownerName,
			AvatarUrl: ownerAvatar,
			Email:     ownerEmail,
		},
		Likes:   []Like{},
		Members: []User{},
	}
	var like *Like
	if likeID.Valid {
		like = &Like{
			ID:        int(likeID.Int32),
			UserID:    likeUID.String,
			EventID:   likeEID.String,
			CreatedAt: likeCreated.String,
		}
	}
	member := &User{
		ID:        mbrID,
		OAuthId:   mbrAuthID,
		Name:      mbrName,
		AvatarUrl: mbrAvatar,
		Email:     mbrEmail,
	}
	return event, like, member, nil
}

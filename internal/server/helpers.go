package server

import "github.com/google/uuid"

func UUID() uuid.UUID {
	u, err := uuid.NewV7()
	if err != nil {
		panic(err)
	}
	return u
}

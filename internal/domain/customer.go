package domain

import "time"

type Customer struct {
	ID        string
	Email     string
	CreatedAt time.Time
}

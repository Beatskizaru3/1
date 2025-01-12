package DB

import "time"

type Task struct {
	ID        *int      `json:"id"`
	TaskName  string    `json:"name"`
	Descr     string    `json:"description"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

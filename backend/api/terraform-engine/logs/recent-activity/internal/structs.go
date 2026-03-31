package internal

import "time"

type DeploymentLog struct {
	DiagramName string    `json:"diagram_name"`
	CommandId   string    `json:"command_id"`
	Command     string    `json:"command"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	Duration    int       `json:"duration_seconds"`
}

package model

import "time"

type RequestChunk struct {
	ID         uint      `json:"id"`
	TraceID    string    `json:"trace_id"`
	ChunkIndex int       `json:"chunk_index"`
	ChunkData  string    `json:"chunk_data"`
	CreatedAt  time.Time `json:"created_at"`
}

package models

type Request struct {
	ID      int         `json:"id"`
	Type    string      `json:"type"`
	Content interface{} `json:"content"`
}

type Response struct {
	ID      int         `json:"id"`
	Type    string      `json:"type"`
	Success bool        `json:"success"`
	Error   string      `json:"error,omitempty"`
	Content interface{} `json:"content,omitempty"`
}

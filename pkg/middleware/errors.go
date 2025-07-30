package middleware

import (
	"net/http"
	"time"
)

// Error represents an specific instance of an error send to the caller. Don't use this directly,
// use a the ErrorFactory function to return the Error.
type Error struct {
	Kind        string    `json:"kind,omitempty"`
	ID          string    `json:"id,omitempty"`
	HREF        string    `json:"href,omitempty"`
	Code        string    `json:"code,omitempty"`
	Reason      string    `json:"reason,omitempty"`
	Details     []any     `json:"details,omitempty"`
	OperationID *string   `json:"operation_id,omitempty"`
	Timestamp   time.Time `json:"timestamp,omitempty"`
}

type SendErrorFunc func(w http.ResponseWriter, r *http.Request, body *Error)

type ErrorFactory func(r *http.Request, format string, a any) Error

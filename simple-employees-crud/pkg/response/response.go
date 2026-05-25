// Package response provides the canonical JSON envelope used by every handler.
// All HTTP responses share the same top-level structure so clients can handle
// success and error cases uniformly.
//
//	Success:  {"success":true,  "data":{...}, "meta":{...}}
//	Error:    {"success":false, "error":{"code":"...","message":"..."}}
package response

import (
	"net/http"

	"simple-employees-crud/pkg/apperror"

	"github.com/gin-gonic/gin"
)

// Envelope is the top-level JSON wrapper for all API responses.
type Envelope struct {
	Success bool        `json:"success"`
	Data    any         `json:"data,omitempty"`
	Error   *ErrorBody  `json:"error,omitempty"`
	Meta    *Pagination `json:"meta,omitempty"`
}

// ErrorBody is the structured error object inside a failed Envelope.
type ErrorBody struct {
	Code    apperror.ErrorCode `json:"code"`
	Message string             `json:"message"`
	Details any                `json:"details,omitempty"`
}

// Pagination carries cursor metadata for list endpoints.
type Pagination struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

// NewPagination builds a Pagination from the raw count and request parameters.
func NewPagination(page, limit int, total int64) *Pagination {
	totalPages := int(total) / limit
	if int(total)%limit != 0 {
		totalPages++
	}
	return &Pagination{
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
	}
}

// ─── Response helpers ────────────────────────────────────────────────────────

// OK writes a 200 response with data.
func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, Envelope{Success: true, Data: data})
}

// OKList writes a 200 response with data and pagination metadata.
func OKList(c *gin.Context, data any, meta *Pagination) {
	c.JSON(http.StatusOK, Envelope{Success: true, Data: data, Meta: meta})
}

// Created writes a 201 response with the newly created resource.
func Created(c *gin.Context, data any) {
	c.JSON(http.StatusCreated, Envelope{Success: true, Data: data})
}

// NoContent writes a 204 response (used for DELETE/logout).
func NoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

// Err writes an error response derived from an *apperror.AppError.
// If err is nil it falls back to a generic 500.
func Err(c *gin.Context, err *apperror.AppError) {
	if err == nil {
		err = apperror.NewInternal("an unexpected error occurred")
	}
	c.JSON(err.StatusCode, Envelope{
		Success: false,
		Error: &ErrorBody{
			Code:    err.Code,
			Message: err.Message,
			Details: err.Details,
		},
	})
}

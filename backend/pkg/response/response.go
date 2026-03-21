// Package response provides standardized HTTP response helpers for the ZenReply API.
package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Response is the standard API response envelope.
type Response struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   *APIError   `json:"error,omitempty"`
	Meta    *Meta       `json:"meta,omitempty"`
}

// APIError represents a structured error response.
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// Meta holds pagination metadata.
type Meta struct {
	Page       int   `json:"page"`
	PerPage    int   `json:"per_page"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

// OK sends a 200 OK response with data.
func OK(c *gin.Context, message string, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Success: true,
		Message: message,
		Data:    data,
	})
}

// Created sends a 201 Created response with data.
func Created(c *gin.Context, message string, data interface{}) {
	c.JSON(http.StatusCreated, Response{
		Success: true,
		Message: message,
		Data:    data,
	})
}

// OKWithMeta sends a 200 OK response with data and pagination metadata.
func OKWithMeta(c *gin.Context, message string, data interface{}, meta *Meta) {
	c.JSON(http.StatusOK, Response{
		Success: true,
		Message: message,
		Data:    data,
		Meta:    meta,
	})
}

// NoContent sends a 204 No Content response.
func NoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

// BadRequest sends a 400 Bad Request error response.
func BadRequest(c *gin.Context, code, message, details string) {
	c.JSON(http.StatusBadRequest, Response{
		Success: false,
		Error: &APIError{
			Code:    code,
			Message: message,
			Details: details,
		},
	})
}

// Unauthorized sends a 401 Unauthorized error response.
func Unauthorized(c *gin.Context, message string) {
	c.JSON(http.StatusUnauthorized, Response{
		Success: false,
		Error: &APIError{
			Code:    "UNAUTHORIZED",
			Message: message,
		},
	})
}

// Forbidden sends a 403 Forbidden error response.
func Forbidden(c *gin.Context, message string) {
	c.JSON(http.StatusForbidden, Response{
		Success: false,
		Error: &APIError{
			Code:    "FORBIDDEN",
			Message: message,
		},
	})
}

// NotFound sends a 404 Not Found error response.
func NotFound(c *gin.Context, message string) {
	c.JSON(http.StatusNotFound, Response{
		Success: false,
		Error: &APIError{
			Code:    "NOT_FOUND",
			Message: message,
		},
	})
}

// Conflict sends a 409 Conflict error response.
func Conflict(c *gin.Context, message string) {
	c.JSON(http.StatusConflict, Response{
		Success: false,
		Error: &APIError{
			Code:    "CONFLICT",
			Message: message,
		},
	})
}

// UnprocessableEntity sends a 422 Unprocessable Entity error response.
func UnprocessableEntity(c *gin.Context, message, details string) {
	c.JSON(http.StatusUnprocessableEntity, Response{
		Success: false,
		Error: &APIError{
			Code:    "VALIDATION_ERROR",
			Message: message,
			Details: details,
		},
	})
}

// InternalServerError sends a 500 Internal Server Error response.
func InternalServerError(c *gin.Context, message string) {
	c.JSON(http.StatusInternalServerError, Response{
		Success: false,
		Error: &APIError{
			Code:    "INTERNAL_SERVER_ERROR",
			Message: message,
		},
	})
}

// ServiceUnavailable sends a 503 Service Unavailable response.
func ServiceUnavailable(c *gin.Context, message string) {
	c.JSON(http.StatusServiceUnavailable, Response{
		Success: false,
		Error: &APIError{
			Code:    "SERVICE_UNAVAILABLE",
			Message: message,
		},
	})
}

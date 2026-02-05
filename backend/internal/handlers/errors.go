package handlers

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// ErrorResponse represents a standardized error response
type ErrorResponse struct {
	Error   string                 `json:"error"`
	Message string                 `json:"message,omitempty"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// respondWithError sends a standardized error response
func respondWithError(c *gin.Context, statusCode int, errorType string, message string) {
	c.JSON(statusCode, ErrorResponse{
		Error:   errorType,
		Message: message,
	})
}

// respondWithValidationError parses validation errors and sends detailed response
func respondWithValidationError(c *gin.Context, err error) {
	details := make(map[string]interface{})

	// Try to parse validator errors
	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		for _, fieldError := range validationErrors {
			fieldName := fieldError.Field()
			switch fieldError.Tag() {
			case "required":
				details[fieldName] = "This field is required"
			case "oneof":
				details[fieldName] = fmt.Sprintf("Must be one of: %s", fieldError.Param())
			case "email":
				details[fieldName] = "Must be a valid email address"
			case "uuid":
				details[fieldName] = "Must be a valid UUID"
			default:
				details[fieldName] = fmt.Sprintf("Validation failed on '%s'", fieldError.Tag())
			}
		}
	} else {
		// Generic validation error
		details["message"] = err.Error()
	}

	c.JSON(http.StatusBadRequest, ErrorResponse{
		Error:   "validation_error",
		Message: "Invalid input",
		Details: details,
	})
}

// respondBadRequest sends a 400 Bad Request error
func respondBadRequest(c *gin.Context, message string) {
	respondWithError(c, http.StatusBadRequest, "bad_request", message)
}

// respondUnauthorized sends a 401 Unauthorized error
func respondUnauthorized(c *gin.Context, message string) {
	respondWithError(c, http.StatusUnauthorized, "unauthorized", message)
}

// respondForbidden sends a 403 Forbidden error
func respondForbidden(c *gin.Context, message string) {
	respondWithError(c, http.StatusForbidden, "forbidden", message)
}

// respondNotFound sends a 404 Not Found error
func respondNotFound(c *gin.Context, resource string) {
	message := fmt.Sprintf("%s not found", resource)
	if resource == "" {
		message = "Resource not found"
	}
	respondWithError(c, http.StatusNotFound, "not_found", message)
}

// respondInternalError sends a 500 Internal Server Error
// In production, you may want to log the actual error but not expose it to clients
func respondInternalError(c *gin.Context, message string, err error) {
	// Log the actual error for debugging (you can replace with proper logging)
	if err != nil {
		fmt.Printf("Internal error: %v\n", err)
	}

	respondWithError(c, http.StatusInternalServerError, "internal_error", message)
}
func respondInvalidID(c *gin.Context, message string) {
	respondWithError(c, http.StatusBadRequest, "invalid_id", message)
}

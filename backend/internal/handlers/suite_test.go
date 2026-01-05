package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/M1ngdaXie/realtime-collab/internal/store"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
)

// BaseHandlerSuite provides common setup for all handler tests
type BaseHandlerSuite struct {
	suite.Suite
	store       *store.PostgresStore
	cleanup     func()
	testData    *store.TestData
	jwtSecret   string
	frontendURL string
}

// SetupSuite runs once before all tests in the suite
func (s *BaseHandlerSuite) SetupSuite() {
	s.store, s.cleanup = store.SetupTestDB(s.T())
	s.jwtSecret = "test-secret-key-do-not-use-in-production"
	s.frontendURL = "http://localhost:5173"
	gin.SetMode(gin.TestMode)
}

// SetupTest runs before each test
func (s *BaseHandlerSuite) SetupTest() {
	ctx := context.Background()
	err := store.TruncateAllTables(ctx, s.store)
	s.Require().NoError(err, "Failed to truncate tables")

	testData, err := store.SeedTestData(ctx, s.store)
	s.Require().NoError(err, "Failed to seed test data")
	s.testData = testData
}

// TearDownSuite runs once after all tests in the suite
func (s *BaseHandlerSuite) TearDownSuite() {
	if s.cleanup != nil {
		s.cleanup()
	}
}

// Helper: makeAuthRequest creates an authenticated HTTP request and returns recorder + request
func (s *BaseHandlerSuite) makeAuthRequest(method, path string, body interface{}, userID uuid.UUID) (*httptest.ResponseRecorder, *http.Request, error) {
	var bodyReader *bytes.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to marshal body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	} else {
		bodyReader = bytes.NewReader([]byte{})
	}

	req := httptest.NewRequest(method, path, bodyReader)
	req.Header.Set("Content-Type", "application/json")

	// Generate JWT token
	token, err := store.GenerateTestJWT(userID, s.jwtSecret)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate JWT: %w", err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	w := httptest.NewRecorder()
	return w, req, nil
}

// Helper: makePublicRequest creates an unauthenticated HTTP request
func (s *BaseHandlerSuite) makePublicRequest(method, path string, body interface{}) (*httptest.ResponseRecorder, *http.Request, error) {
	var bodyReader *bytes.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to marshal body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	} else {
		bodyReader = bytes.NewReader([]byte{})
	}

	req := httptest.NewRequest(method, path, bodyReader)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	return w, req, nil
}

// Helper: parseErrorResponse parses an ErrorResponse from the response
func (s *BaseHandlerSuite) parseErrorResponse(w *httptest.ResponseRecorder) ErrorResponse {
	var errResp ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &errResp)
	s.Require().NoError(err, "Failed to parse error response")
	return errResp
}

// Helper: assertErrorResponse checks that the response matches expected error
func (s *BaseHandlerSuite) assertErrorResponse(w *httptest.ResponseRecorder, statusCode int, errorType string, messageContains string) {
	s.Equal(statusCode, w.Code, "Status code mismatch")

	errResp := s.parseErrorResponse(w)
	s.Equal(errorType, errResp.Error, "Error type mismatch")

	if messageContains != "" {
		s.Contains(errResp.Message, messageContains, "Error message should contain expected text")
	}
}

// Helper: assertSuccessResponse checks that the response is successful
func (s *BaseHandlerSuite) assertSuccessResponse(w *httptest.ResponseRecorder, statusCode int) {
	s.Equal(statusCode, w.Code, "Status code mismatch. Body: %s", w.Body.String())
}

// Helper: parseJSONResponse parses a JSON response into the target struct
func (s *BaseHandlerSuite) parseJSONResponse(w *httptest.ResponseRecorder, target interface{}) {
	err := json.Unmarshal(w.Body.Bytes(), target)
	s.Require().NoError(err, "Failed to parse JSON response: %s", w.Body.String())
}

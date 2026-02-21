package user_suggestions

import (
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/bootstrap"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type mockRepository struct {
	// Configurable return values
	allWithScores    []SuggestionWithScore
	allWithScoresErr error
	suggestion       *model.UserSuggestions
	suggestionErr    error
	created          *model.UserSuggestions
	createErr        error
	updated          *model.UserSuggestions
	updateErr        error
	deleteErr        error
	upsertVoteErr    error
	deleteVoteErr    error

	// Capture calls
	capturedVoterID      string
	capturedSuggestionID uuid.UUID
	capturedDirection    int16
	capturedDeleteID     uuid.UUID
}

func (m *mockRepository) GetAllWithScores(voterID string) ([]SuggestionWithScore, error) {
	m.capturedVoterID = voterID
	return m.allWithScores, m.allWithScoresErr
}

func (m *mockRepository) GetByID(id uuid.UUID) (*model.UserSuggestions, error) {
	return m.suggestion, m.suggestionErr
}

func (m *mockRepository) Create(suggestion model.UserSuggestions) (*model.UserSuggestions, error) {
	if m.created != nil {
		return m.created, m.createErr
	}
	suggestion.ID = uuid.New()
	suggestion.CreatedAt = time.Now()
	return &suggestion, m.createErr
}

func (m *mockRepository) Update(id uuid.UUID, title, description string) (*model.UserSuggestions, error) {
	return m.updated, m.updateErr
}

func (m *mockRepository) Delete(id uuid.UUID) error {
	m.capturedDeleteID = id
	return m.deleteErr
}

func (m *mockRepository) UpsertVote(suggestionID uuid.UUID, voterID string, direction int16) error {
	m.capturedSuggestionID = suggestionID
	m.capturedDirection = direction
	return m.upsertVoteErr
}

func (m *mockRepository) DeleteVote(suggestionID uuid.UUID, voterID string) error {
	m.capturedSuggestionID = suggestionID
	return m.deleteVoteErr
}

// --- CreateSuggestion tests ---

func TestCreateSuggestion_Unauthorized(t *testing.T) {
	repo := &mockRepository{}
	svc := NewService(repo)

	_, err := svc.CreateSuggestion(nil, "Title", "Description")
	require.ErrorIs(t, err, ErrUnauthorized)
}

func TestCreateSuggestion_InvalidTitle_Empty(t *testing.T) {
	repo := &mockRepository{}
	svc := NewService(repo)
	user := &bootstrap.User{UserID: "user-1", Username: "test"}

	_, err := svc.CreateSuggestion(user, "", "Description")
	require.ErrorIs(t, err, ErrInvalidTitle)
}

func TestCreateSuggestion_InvalidTitle_TooLong(t *testing.T) {
	repo := &mockRepository{}
	svc := NewService(repo)
	user := &bootstrap.User{UserID: "user-1", Username: "test"}

	longTitle := make([]byte, 201)
	for i := range longTitle {
		longTitle[i] = 'a'
	}
	_, err := svc.CreateSuggestion(user, string(longTitle), "Description")
	require.ErrorIs(t, err, ErrInvalidTitle)
}

func TestCreateSuggestion_InvalidDescription_Empty(t *testing.T) {
	repo := &mockRepository{}
	svc := NewService(repo)
	user := &bootstrap.User{UserID: "user-1", Username: "test"}

	_, err := svc.CreateSuggestion(user, "Title", "")
	require.ErrorIs(t, err, ErrInvalidDescription)
}

func TestCreateSuggestion_Success(t *testing.T) {
	repo := &mockRepository{}
	svc := NewService(repo)
	user := &bootstrap.User{UserID: "user-1", Username: "testuser"}

	result, err := svc.CreateSuggestion(user, "My Feature", "A great feature")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "My Feature", result.Title)
	require.Equal(t, "A great feature", result.Description)
	require.Equal(t, "testuser", result.Username)
	require.Equal(t, "user-1", result.UserID)
}

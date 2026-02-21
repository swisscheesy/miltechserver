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

// --- UpdateSuggestion tests ---

func TestUpdateSuggestion_Unauthorized(t *testing.T) {
	repo := &mockRepository{}
	svc := NewService(repo)

	_, err := svc.UpdateSuggestion(nil, uuid.New().String(), "Title", "Desc")
	require.ErrorIs(t, err, ErrUnauthorized)
}

func TestUpdateSuggestion_InvalidID(t *testing.T) {
	repo := &mockRepository{}
	svc := NewService(repo)
	user := &bootstrap.User{UserID: "user-1", Username: "test"}

	_, err := svc.UpdateSuggestion(user, "not-a-uuid", "Title", "Desc")
	require.ErrorIs(t, err, ErrInvalidID)
}

func TestUpdateSuggestion_NotFound(t *testing.T) {
	repo := &mockRepository{suggestion: nil}
	svc := NewService(repo)
	user := &bootstrap.User{UserID: "user-1", Username: "test"}

	_, err := svc.UpdateSuggestion(user, uuid.New().String(), "Title", "Desc")
	require.ErrorIs(t, err, ErrSuggestionNotFound)
}

func TestUpdateSuggestion_Forbidden(t *testing.T) {
	suggID := uuid.New()
	repo := &mockRepository{
		suggestion: &model.UserSuggestions{
			ID:     suggID,
			UserID: "user-2",
			Title:  "Old Title",
		},
	}
	svc := NewService(repo)
	user := &bootstrap.User{UserID: "user-1", Username: "test"}

	_, err := svc.UpdateSuggestion(user, suggID.String(), "New Title", "New Desc")
	require.ErrorIs(t, err, ErrForbidden)
}

func TestUpdateSuggestion_InvalidTitle(t *testing.T) {
	suggID := uuid.New()
	repo := &mockRepository{
		suggestion: &model.UserSuggestions{
			ID:     suggID,
			UserID: "user-1",
		},
	}
	svc := NewService(repo)
	user := &bootstrap.User{UserID: "user-1", Username: "test"}

	_, err := svc.UpdateSuggestion(user, suggID.String(), "", "Desc")
	require.ErrorIs(t, err, ErrInvalidTitle)
}

func TestUpdateSuggestion_Success(t *testing.T) {
	suggID := uuid.New()
	now := time.Now()
	repo := &mockRepository{
		suggestion: &model.UserSuggestions{
			ID:     suggID,
			UserID: "user-1",
			Title:  "Old",
			Status: "Submitted",
		},
		updated: &model.UserSuggestions{
			ID:          suggID,
			UserID:      "user-1",
			Title:       "New Title",
			Description: "New Desc",
			Status:      "Submitted",
			CreatedAt:   now,
			UpdatedAt:   &now,
		},
	}
	svc := NewService(repo)
	user := &bootstrap.User{UserID: "user-1", Username: "testuser"}

	result, err := svc.UpdateSuggestion(user, suggID.String(), "New Title", "New Desc")
	require.NoError(t, err)
	require.Equal(t, "New Title", result.Title)
	require.Equal(t, "New Desc", result.Description)
}

// --- DeleteSuggestion tests ---

func TestDeleteSuggestion_Unauthorized(t *testing.T) {
	repo := &mockRepository{}
	svc := NewService(repo)

	err := svc.DeleteSuggestion(nil, uuid.New().String())
	require.ErrorIs(t, err, ErrUnauthorized)
}

func TestDeleteSuggestion_InvalidID(t *testing.T) {
	repo := &mockRepository{}
	svc := NewService(repo)
	user := &bootstrap.User{UserID: "user-1", Username: "test"}

	err := svc.DeleteSuggestion(user, "not-a-uuid")
	require.ErrorIs(t, err, ErrInvalidID)
}

func TestDeleteSuggestion_NotFound(t *testing.T) {
	repo := &mockRepository{suggestion: nil}
	svc := NewService(repo)
	user := &bootstrap.User{UserID: "user-1", Username: "test"}

	err := svc.DeleteSuggestion(user, uuid.New().String())
	require.ErrorIs(t, err, ErrSuggestionNotFound)
}

func TestDeleteSuggestion_Forbidden(t *testing.T) {
	suggID := uuid.New()
	repo := &mockRepository{
		suggestion: &model.UserSuggestions{
			ID:     suggID,
			UserID: "user-2",
		},
	}
	svc := NewService(repo)
	user := &bootstrap.User{UserID: "user-1", Username: "test"}

	err := svc.DeleteSuggestion(user, suggID.String())
	require.ErrorIs(t, err, ErrForbidden)
}

func TestDeleteSuggestion_Success(t *testing.T) {
	suggID := uuid.New()
	repo := &mockRepository{
		suggestion: &model.UserSuggestions{
			ID:     suggID,
			UserID: "user-1",
		},
	}
	svc := NewService(repo)
	user := &bootstrap.User{UserID: "user-1", Username: "test"}

	err := svc.DeleteSuggestion(user, suggID.String())
	require.NoError(t, err)
	require.Equal(t, suggID, repo.capturedDeleteID)
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

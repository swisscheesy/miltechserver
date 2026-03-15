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
	existingVote     *int16
	existingVoteErr  error
	upsertVoteErr    error
	deleteVoteErr    error

	// Capture calls
	capturedVoterID          string
	capturedSuggestionID     uuid.UUID
	capturedDirection        int16
	capturedDeleteID         uuid.UUID
	capturedCreateSuggestion model.UserSuggestions
	upsertVoteCalled         bool
	deleteVoteCalled         bool
}

func (m *mockRepository) GetAllWithScores(voterID string) ([]SuggestionWithScore, error) {
	m.capturedVoterID = voterID
	return m.allWithScores, m.allWithScoresErr
}

func (m *mockRepository) GetByID(id uuid.UUID) (*model.UserSuggestions, error) {
	return m.suggestion, m.suggestionErr
}

func (m *mockRepository) Create(suggestion model.UserSuggestions) (*model.UserSuggestions, error) {
	m.capturedCreateSuggestion = suggestion
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

func (m *mockRepository) GetVote(suggestionID uuid.UUID, voterID string) (*int16, error) {
	return m.existingVote, m.existingVoteErr
}

func (m *mockRepository) UpsertVote(suggestionID uuid.UUID, voterID string, direction int16) error {
	m.capturedSuggestionID = suggestionID
	m.capturedDirection = direction
	m.upsertVoteCalled = true
	return m.upsertVoteErr
}

func (m *mockRepository) DeleteVote(suggestionID uuid.UUID, voterID string) error {
	m.capturedSuggestionID = suggestionID
	m.deleteVoteCalled = true
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

// --- Vote tests ---

func TestVote_Unauthorized(t *testing.T) {
	repo := &mockRepository{}
	svc := NewService(repo)

	err := svc.Vote(nil, uuid.New().String(), 1)
	require.ErrorIs(t, err, ErrUnauthorized)
}

func TestVote_InvalidID(t *testing.T) {
	repo := &mockRepository{}
	svc := NewService(repo)
	user := &bootstrap.User{UserID: "user-1", Username: "test"}

	err := svc.Vote(user, "not-a-uuid", 1)
	require.ErrorIs(t, err, ErrInvalidID)
}

func TestVote_InvalidDirection(t *testing.T) {
	suggID := uuid.New()
	repo := &mockRepository{
		suggestion: &model.UserSuggestions{ID: suggID, UserID: "user-2"},
	}
	svc := NewService(repo)
	user := &bootstrap.User{UserID: "user-1", Username: "test"}

	err := svc.Vote(user, suggID.String(), 0)
	require.ErrorIs(t, err, ErrInvalidDirection)

	err = svc.Vote(user, suggID.String(), 2)
	require.ErrorIs(t, err, ErrInvalidDirection)
}

func TestVote_SuggestionNotFound(t *testing.T) {
	repo := &mockRepository{suggestion: nil}
	svc := NewService(repo)
	user := &bootstrap.User{UserID: "user-1", Username: "test"}

	err := svc.Vote(user, uuid.New().String(), 1)
	require.ErrorIs(t, err, ErrSuggestionNotFound)
}

func TestVote_Upvote(t *testing.T) {
	suggID := uuid.New()
	repo := &mockRepository{
		suggestion: &model.UserSuggestions{ID: suggID, UserID: "user-2"},
	}
	svc := NewService(repo)
	user := &bootstrap.User{UserID: "user-1", Username: "test"}

	err := svc.Vote(user, suggID.String(), 1)
	require.NoError(t, err)
	require.Equal(t, suggID, repo.capturedSuggestionID)
	require.Equal(t, int16(1), repo.capturedDirection)
}

func TestVote_Downvote(t *testing.T) {
	suggID := uuid.New()
	repo := &mockRepository{
		suggestion: &model.UserSuggestions{ID: suggID, UserID: "user-2"},
	}
	svc := NewService(repo)
	user := &bootstrap.User{UserID: "user-1", Username: "test"}

	err := svc.Vote(user, suggID.String(), -1)
	require.NoError(t, err)
	require.Equal(t, int16(-1), repo.capturedDirection)
}

// --- RemoveVote tests ---

func TestRemoveVote_Unauthorized(t *testing.T) {
	repo := &mockRepository{}
	svc := NewService(repo)

	err := svc.RemoveVote(nil, uuid.New().String())
	require.ErrorIs(t, err, ErrUnauthorized)
}

func TestRemoveVote_Success(t *testing.T) {
	suggID := uuid.New()
	repo := &mockRepository{}
	svc := NewService(repo)
	user := &bootstrap.User{UserID: "user-1", Username: "test"}

	err := svc.RemoveVote(user, suggID.String())
	require.NoError(t, err)
	require.Equal(t, suggID, repo.capturedSuggestionID)
}

// --- GetAllSuggestions tests ---

func strPtr(s string) *string {
	return &s
}

func TestGetAllSuggestions_Unauthenticated(t *testing.T) {
	repo := &mockRepository{
		allWithScores: []SuggestionWithScore{
			{
				ID:          uuid.New(),
				UserID:      "user-1",
				Title:       "Feature A",
				Description: "Description A",
				Status:      "Submitted",
				CreatedAt:   time.Now(),
				Username:    strPtr("testuser"),
				Score:       5,
				MyVote:      nil,
			},
		},
	}
	svc := NewService(repo)

	results, err := svc.GetAllSuggestions(nil)
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Equal(t, "Feature A", results[0].Title)
	require.Equal(t, 5, results[0].Score)
	require.Nil(t, results[0].MyVote)
	require.Equal(t, "", repo.capturedVoterID)
}

func TestGetAllSuggestions_Authenticated(t *testing.T) {
	vote := int16(1)
	repo := &mockRepository{
		allWithScores: []SuggestionWithScore{
			{
				ID:          uuid.New(),
				UserID:      "user-1",
				Title:       "Feature A",
				Description: "Description A",
				Status:      "Submitted",
				CreatedAt:   time.Now(),
				Username:    strPtr("testuser"),
				Score:       3,
				MyVote:      &vote,
			},
		},
	}
	svc := NewService(repo)
	user := &bootstrap.User{UserID: "user-1", Username: "testuser"}

	results, err := svc.GetAllSuggestions(user)
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Equal(t, int16(1), *results[0].MyVote)
	require.Equal(t, "user-1", repo.capturedVoterID)
}

func TestGetAllSuggestions_Empty(t *testing.T) {
	repo := &mockRepository{
		allWithScores: []SuggestionWithScore{},
	}
	svc := NewService(repo)

	results, err := svc.GetAllSuggestions(nil)
	require.NoError(t, err)
	require.Empty(t, results)
}

// --- Vote toggle behavior tests ---

func int16Ptr(v int16) *int16 {
	return &v
}

func TestVote_AddsVote_WhenNoExistingVote(t *testing.T) {
	suggID := uuid.New()
	repo := &mockRepository{
		suggestion:   &model.UserSuggestions{ID: suggID, UserID: "user-2"},
		existingVote: nil,
	}
	svc := NewService(repo)
	user := &bootstrap.User{UserID: "user-1", Username: "test"}

	err := svc.Vote(user, suggID.String(), 1)
	require.NoError(t, err)
	require.True(t, repo.upsertVoteCalled, "UpsertVote should be called for new vote")
	require.False(t, repo.deleteVoteCalled, "DeleteVote should not be called for new vote")
	require.Equal(t, int16(1), repo.capturedDirection)
}

func TestVote_TogglesOff_WhenExistingVoteSameDirection(t *testing.T) {
	suggID := uuid.New()
	repo := &mockRepository{
		suggestion:   &model.UserSuggestions{ID: suggID, UserID: "user-2"},
		existingVote: int16Ptr(1),
	}
	svc := NewService(repo)
	user := &bootstrap.User{UserID: "user-1", Username: "test"}

	err := svc.Vote(user, suggID.String(), 1)
	require.NoError(t, err)
	require.True(t, repo.deleteVoteCalled, "DeleteVote should be called to toggle off")
	require.False(t, repo.upsertVoteCalled, "UpsertVote should not be called when toggling off")
}

func TestVote_TogglesOff_WhenExistingVoteOppositeDirection(t *testing.T) {
	suggID := uuid.New()
	repo := &mockRepository{
		suggestion:   &model.UserSuggestions{ID: suggID, UserID: "user-2"},
		existingVote: int16Ptr(1),
	}
	svc := NewService(repo)
	user := &bootstrap.User{UserID: "user-1", Username: "test"}

	err := svc.Vote(user, suggID.String(), -1)
	require.NoError(t, err)
	require.True(t, repo.deleteVoteCalled, "DeleteVote should be called when switching direction")
	require.False(t, repo.upsertVoteCalled, "UpsertVote should not be called when switching direction")
}

func TestCreateSuggestion_SetsShowFalse(t *testing.T) {
	repo := &mockRepository{}
	svc := NewService(repo)
	user := &bootstrap.User{UserID: "user-1", Username: "testuser"}

	_, err := svc.CreateSuggestion(user, "My Feature", "A great feature")
	require.NoError(t, err)
	require.NotNil(t, repo.capturedCreateSuggestion.Show, "Show must be explicitly set, not nil")
	require.False(t, *repo.capturedCreateSuggestion.Show, "newly created suggestions must have show=false")
}

func TestVote_TogglesOff_WhenExistingDownvoteAndUpvoteRequested(t *testing.T) {
	suggID := uuid.New()
	repo := &mockRepository{
		suggestion:   &model.UserSuggestions{ID: suggID, UserID: "user-2"},
		existingVote: int16Ptr(-1),
	}
	svc := NewService(repo)
	user := &bootstrap.User{UserID: "user-1", Username: "test"}

	err := svc.Vote(user, suggID.String(), 1)
	require.NoError(t, err)
	require.True(t, repo.deleteVoteCalled, "DeleteVote should be called when switching from downvote to upvote")
	require.False(t, repo.upsertVoteCalled, "UpsertVote should not be called when switching direction")
}

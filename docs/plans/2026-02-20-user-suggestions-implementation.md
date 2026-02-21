# User Suggestions Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement a user suggestion system with submit, vote, edit, delete, and list operations.

**Architecture:** Standalone `api/user_suggestions/` domain package following the `item_comments` flat-package pattern. Two Postgres tables (`user_suggestions`, `user_suggestion_votes`), a Repository/Service/Handler layered architecture with TDD, and a new optional auth middleware for the public GET endpoint.

**Tech Stack:** Go 1.21+, Gin, Jet (code generation + query building), PostgreSQL, Firebase Auth, testify

**Design doc:** `docs/plans/2026-02-20-user-suggestions-design.md`

---

### Task 1: Database Migration

Create the two new Postgres tables.

**Files:**
- Create: SQL migration (apply directly to database)

**Step 1: Write the migration SQL**

```sql
-- user_suggestions table
CREATE TABLE user_suggestions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     TEXT NOT NULL,
    title       TEXT NOT NULL CHECK (length(title) BETWEEN 1 AND 200),
    description TEXT NOT NULL CHECK (length(description) BETWEEN 1 AND 2000),
    status      TEXT NOT NULL DEFAULT 'Submitted',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ
);

CREATE INDEX idx_user_suggestions_user_id ON user_suggestions (user_id);
CREATE INDEX idx_user_suggestions_created_at ON user_suggestions (created_at);

-- user_suggestion_votes table
CREATE TABLE user_suggestion_votes (
    suggestion_id UUID NOT NULL REFERENCES user_suggestions(id) ON DELETE CASCADE,
    voter_id      TEXT NOT NULL,
    direction     SMALLINT NOT NULL CHECK (direction IN (-1, 1)),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (suggestion_id, voter_id)
);
```

**Step 2: Apply migration to database**

Run the migration against the Postgres database. Verify tables exist:

```bash
psql -c "\d user_suggestions"
psql -c "\d user_suggestion_votes"
```

Expected: Both tables shown with correct columns, constraints, and indexes.

**Step 3: Regenerate Jet models**

Run Jet code generation to create Go model/table files for the new tables:

```bash
# From project root — use whatever Jet generation command the project uses
# Check Makefile or scripts for the exact command
go generate ./...
```

Verify new files appear:
- `.gen/miltech_ng/public/model/user_suggestions.go`
- `.gen/miltech_ng/public/model/user_suggestion_votes.go`
- `.gen/miltech_ng/public/table/user_suggestions.go`
- `.gen/miltech_ng/public/table/user_suggestion_votes.go`

**Step 4: Commit**

```bash
git add .gen/miltech_ng/public/model/user_suggestions.go \
       .gen/miltech_ng/public/model/user_suggestion_votes.go \
       .gen/miltech_ng/public/table/user_suggestions.go \
       .gen/miltech_ng/public/table/user_suggestion_votes.go
git commit -m "feat(user-suggestions): add database tables and Jet models"
```

---

### Task 2: Foundation Files — Errors and Types

Create the sentinel errors and request/response types.

**Files:**
- Create: `api/user_suggestions/errors.go`
- Create: `api/user_suggestions/types.go`

**Step 1: Create errors.go**

```go
package user_suggestions

import "errors"

var (
	ErrUnauthorized       = errors.New("unauthorized user")
	ErrForbidden          = errors.New("not authorized to modify this suggestion")
	ErrSuggestionNotFound = errors.New("suggestion not found")
	ErrInvalidTitle       = errors.New("title must be between 1 and 200 characters")
	ErrInvalidDescription = errors.New("description must be between 1 and 2000 characters")
	ErrInvalidDirection   = errors.New("vote direction must be 1 or -1")
	ErrInvalidID          = errors.New("invalid suggestion ID")
)
```

**Step 2: Create types.go**

```go
package user_suggestions

import (
	"time"

	"github.com/google/uuid"
)

// --- Requests ---

type CreateSuggestionRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

type UpdateSuggestionRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

type VoteRequest struct {
	Direction int16 `json:"direction"`
}

// --- Responses ---

type SuggestionResponse struct {
	ID          string  `json:"id"`
	UserID      string  `json:"user_id"`
	Username    string  `json:"username"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Status      string  `json:"status"`
	Score       int     `json:"score"`
	MyVote      *int16  `json:"my_vote,omitempty"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   *string `json:"updated_at,omitempty"`
}

// --- Internal types ---

type SuggestionWithScore struct {
	ID          uuid.UUID
	UserID      string
	Title       string
	Description string
	Status      string
	CreatedAt   time.Time
	UpdatedAt   *time.Time
	Username    *string
	Score       int
	MyVote      *int16
}
```

**Step 3: Verify it compiles**

```bash
go build ./api/user_suggestions/...
```

Expected: Clean build (no errors).

**Step 4: Commit**

```bash
git add api/user_suggestions/errors.go api/user_suggestions/types.go
git commit -m "feat(user-suggestions): add foundation types and sentinel errors"
```

---

### Task 3: Repository and Service Interfaces

Define the contracts for Repository and Service.

**Files:**
- Create: `api/user_suggestions/repository.go`
- Create: `api/user_suggestions/service.go`

**Step 1: Create repository.go**

```go
package user_suggestions

import (
	"miltechserver/.gen/miltech_ng/public/model"

	"github.com/google/uuid"
)

type Repository interface {
	GetAllWithScores(voterID string) ([]SuggestionWithScore, error)
	GetByID(id uuid.UUID) (*model.UserSuggestions, error)
	Create(suggestion model.UserSuggestions) (*model.UserSuggestions, error)
	Update(id uuid.UUID, title, description string) (*model.UserSuggestions, error)
	Delete(id uuid.UUID) error
	UpsertVote(suggestionID uuid.UUID, voterID string, direction int16) error
	DeleteVote(suggestionID uuid.UUID, voterID string) error
}
```

Note: `model.UserSuggestions` comes from the Jet-generated code in Task 1. If Jet hasn't been run yet, this won't compile — Task 1 must complete first.

**Step 2: Create service.go**

```go
package user_suggestions

import "miltechserver/bootstrap"

type Service interface {
	GetAllSuggestions(currentUser *bootstrap.User) ([]SuggestionResponse, error)
	CreateSuggestion(user *bootstrap.User, title, description string) (*SuggestionResponse, error)
	UpdateSuggestion(user *bootstrap.User, suggestionID, title, description string) (*SuggestionResponse, error)
	DeleteSuggestion(user *bootstrap.User, suggestionID string) error
	Vote(user *bootstrap.User, suggestionID string, direction int16) error
	RemoveVote(user *bootstrap.User, suggestionID string) error
}
```

**Step 3: Verify it compiles**

```bash
go build ./api/user_suggestions/...
```

Expected: Clean build.

**Step 4: Commit**

```bash
git add api/user_suggestions/repository.go api/user_suggestions/service.go
git commit -m "feat(user-suggestions): add repository and service interfaces"
```

---

### Task 4: Service Implementation — Test-Driven (Validation & CRUD)

Build the `ServiceImpl` using TDD. This is the core business logic. We start by creating a mock repository, then writing failing tests, then making them pass.

**Files:**
- Create: `api/user_suggestions/service_impl.go`
- Create: `api/user_suggestions/service_impl_test.go`

**Step 1: Create the mock repository in the test file**

Write `service_impl_test.go` with the mock repository struct that implements the `Repository` interface. This mock captures method calls and returns configurable values:

```go
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
	capturedVoterID     string
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
```

**Step 2: Write failing tests for CreateSuggestion validation**

Add these tests to `service_impl_test.go`:

```go
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
```

**Step 3: Run tests to verify they fail**

```bash
go test ./api/user_suggestions/... -v -run TestCreate
```

Expected: FAIL — `NewService` not defined.

**Step 4: Write ServiceImpl with CreateSuggestion**

Create `service_impl.go`:

```go
package user_suggestions

import (
	"strings"
	"time"

	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/bootstrap"

	"github.com/google/uuid"
)

type ServiceImpl struct {
	repo Repository
}

func NewService(repo Repository) *ServiceImpl {
	return &ServiceImpl{repo: repo}
}

func (s *ServiceImpl) CreateSuggestion(user *bootstrap.User, title, description string) (*SuggestionResponse, error) {
	if user == nil {
		return nil, ErrUnauthorized
	}

	trimmedTitle := strings.TrimSpace(title)
	if len(trimmedTitle) == 0 || len(trimmedTitle) > 200 {
		return nil, ErrInvalidTitle
	}

	trimmedDesc := strings.TrimSpace(description)
	if len(trimmedDesc) == 0 || len(trimmedDesc) > 2000 {
		return nil, ErrInvalidDescription
	}

	suggestion := model.UserSuggestions{
		UserID:      user.UserID,
		Title:       trimmedTitle,
		Description: trimmedDesc,
		Status:      "Submitted",
	}

	created, err := s.repo.Create(suggestion)
	if err != nil {
		return nil, err
	}

	resp := mapSuggestionToResponse(created, user.Username, 0, nil)
	return &resp, nil
}

func mapSuggestionToResponse(s *model.UserSuggestions, username string, score int, myVote *int16) SuggestionResponse {
	resp := SuggestionResponse{
		ID:          s.ID.String(),
		UserID:      s.UserID,
		Username:    username,
		Title:       s.Title,
		Description: s.Description,
		Status:      s.Status,
		Score:       score,
		MyVote:      myVote,
		CreatedAt:   s.CreatedAt.Format(time.RFC3339),
	}

	if s.UpdatedAt != nil {
		formatted := s.UpdatedAt.Format(time.RFC3339)
		resp.UpdatedAt = &formatted
	}

	return resp
}
```

Add placeholder methods to satisfy the interface (we'll implement them in subsequent steps):

```go
func (s *ServiceImpl) GetAllSuggestions(currentUser *bootstrap.User) ([]SuggestionResponse, error) {
	return nil, nil
}

func (s *ServiceImpl) UpdateSuggestion(user *bootstrap.User, suggestionID, title, description string) (*SuggestionResponse, error) {
	return nil, nil
}

func (s *ServiceImpl) DeleteSuggestion(user *bootstrap.User, suggestionID string) error {
	return nil
}

func (s *ServiceImpl) Vote(user *bootstrap.User, suggestionID string, direction int16) error {
	return nil
}

func (s *ServiceImpl) RemoveVote(user *bootstrap.User, suggestionID string) error {
	return nil
}
```

**Step 5: Run tests to verify they pass**

```bash
go test ./api/user_suggestions/... -v -run TestCreate
```

Expected: All 5 tests PASS.

**Step 6: Commit**

```bash
git add api/user_suggestions/service_impl.go api/user_suggestions/service_impl_test.go
git commit -m "feat(user-suggestions): implement CreateSuggestion with TDD"
```

---

### Task 5: Service Implementation — Update and Delete (TDD)

Continue TDD for UpdateSuggestion and DeleteSuggestion.

**Files:**
- Modify: `api/user_suggestions/service_impl.go`
- Modify: `api/user_suggestions/service_impl_test.go`

**Step 1: Write failing tests for UpdateSuggestion**

Add to `service_impl_test.go`:

```go
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
```

**Step 2: Run tests to verify they fail**

```bash
go test ./api/user_suggestions/... -v -run TestUpdate
```

Expected: Most FAIL (placeholder returns nil).

**Step 3: Implement UpdateSuggestion**

Replace the placeholder in `service_impl.go`:

```go
func (s *ServiceImpl) UpdateSuggestion(user *bootstrap.User, suggestionID, title, description string) (*SuggestionResponse, error) {
	if user == nil {
		return nil, ErrUnauthorized
	}

	id, err := uuid.Parse(suggestionID)
	if err != nil {
		return nil, ErrInvalidID
	}

	existing, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, ErrSuggestionNotFound
	}
	if existing.UserID != user.UserID {
		return nil, ErrForbidden
	}

	trimmedTitle := strings.TrimSpace(title)
	if len(trimmedTitle) == 0 || len(trimmedTitle) > 200 {
		return nil, ErrInvalidTitle
	}

	trimmedDesc := strings.TrimSpace(description)
	if len(trimmedDesc) == 0 || len(trimmedDesc) > 2000 {
		return nil, ErrInvalidDescription
	}

	updated, err := s.repo.Update(id, trimmedTitle, trimmedDesc)
	if err != nil {
		return nil, err
	}

	resp := mapSuggestionToResponse(updated, user.Username, 0, nil)
	return &resp, nil
}
```

**Step 4: Write failing tests for DeleteSuggestion**

```go
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
```

**Step 5: Implement DeleteSuggestion**

Replace the placeholder in `service_impl.go`:

```go
func (s *ServiceImpl) DeleteSuggestion(user *bootstrap.User, suggestionID string) error {
	if user == nil {
		return ErrUnauthorized
	}

	id, err := uuid.Parse(suggestionID)
	if err != nil {
		return ErrInvalidID
	}

	existing, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}
	if existing == nil {
		return ErrSuggestionNotFound
	}
	if existing.UserID != user.UserID {
		return ErrForbidden
	}

	return s.repo.Delete(id)
}
```

**Step 6: Run all tests to verify they pass**

```bash
go test ./api/user_suggestions/... -v -run "TestUpdate|TestDelete"
```

Expected: All PASS.

**Step 7: Commit**

```bash
git add api/user_suggestions/service_impl.go api/user_suggestions/service_impl_test.go
git commit -m "feat(user-suggestions): implement UpdateSuggestion and DeleteSuggestion with TDD"
```

---

### Task 6: Service Implementation — Voting and GetAll (TDD)

Continue TDD for Vote, RemoveVote, and GetAllSuggestions.

**Files:**
- Modify: `api/user_suggestions/service_impl.go`
- Modify: `api/user_suggestions/service_impl_test.go`

**Step 1: Write failing tests for Vote**

```go
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

func TestRemoveVote_Success(t *testing.T) {
	suggID := uuid.New()
	repo := &mockRepository{}
	svc := NewService(repo)
	user := &bootstrap.User{UserID: "user-1", Username: "test"}

	err := svc.RemoveVote(user, suggID.String())
	require.NoError(t, err)
	require.Equal(t, suggID, repo.capturedSuggestionID)
}

func TestRemoveVote_Unauthorized(t *testing.T) {
	repo := &mockRepository{}
	svc := NewService(repo)

	err := svc.RemoveVote(nil, uuid.New().String())
	require.ErrorIs(t, err, ErrUnauthorized)
}
```

**Step 2: Run tests to verify they fail**

```bash
go test ./api/user_suggestions/... -v -run "TestVote|TestRemoveVote"
```

Expected: FAIL (placeholders return nil).

**Step 3: Implement Vote and RemoveVote**

Replace placeholders in `service_impl.go`:

```go
func (s *ServiceImpl) Vote(user *bootstrap.User, suggestionID string, direction int16) error {
	if user == nil {
		return ErrUnauthorized
	}

	id, err := uuid.Parse(suggestionID)
	if err != nil {
		return ErrInvalidID
	}

	if direction != 1 && direction != -1 {
		return ErrInvalidDirection
	}

	existing, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}
	if existing == nil {
		return ErrSuggestionNotFound
	}

	return s.repo.UpsertVote(id, user.UserID, direction)
}

func (s *ServiceImpl) RemoveVote(user *bootstrap.User, suggestionID string) error {
	if user == nil {
		return ErrUnauthorized
	}

	id, err := uuid.Parse(suggestionID)
	if err != nil {
		return ErrInvalidID
	}

	return s.repo.DeleteVote(id, user.UserID)
}
```

**Step 4: Write tests for GetAllSuggestions**

```go
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

func strPtr(s string) *string {
	return &s
}
```

**Step 5: Implement GetAllSuggestions**

Replace placeholder:

```go
func (s *ServiceImpl) GetAllSuggestions(currentUser *bootstrap.User) ([]SuggestionResponse, error) {
	voterID := ""
	if currentUser != nil {
		voterID = currentUser.UserID
	}

	suggestions, err := s.repo.GetAllWithScores(voterID)
	if err != nil {
		return nil, err
	}

	results := make([]SuggestionResponse, 0, len(suggestions))
	for _, sug := range suggestions {
		username := "Unknown"
		if sug.Username != nil {
			username = *sug.Username
		}

		resp := SuggestionResponse{
			ID:          sug.ID.String(),
			UserID:      sug.UserID,
			Username:    username,
			Title:       sug.Title,
			Description: sug.Description,
			Status:      sug.Status,
			Score:       sug.Score,
			CreatedAt:   sug.CreatedAt.Format(time.RFC3339),
		}

		if sug.UpdatedAt != nil {
			formatted := sug.UpdatedAt.Format(time.RFC3339)
			resp.UpdatedAt = &formatted
		}

		// Only include MyVote when user is authenticated
		if currentUser != nil {
			resp.MyVote = sug.MyVote
		}

		results = append(results, resp)
	}

	return results, nil
}
```

**Step 6: Run all service tests**

```bash
go test ./api/user_suggestions/... -v
```

Expected: All PASS.

**Step 7: Commit**

```bash
git add api/user_suggestions/service_impl.go api/user_suggestions/service_impl_test.go
git commit -m "feat(user-suggestions): implement Vote, RemoveVote, GetAllSuggestions with TDD"
```

---

### Task 7: Optional Auth Middleware

Create a new middleware that attempts authentication but does not abort on failure.

**Files:**
- Create: `api/middleware/optional_auth.go`

**Step 1: Create optional_auth.go**

```go
package middleware

import (
	"context"
	"log/slog"
	"miltechserver/bootstrap"
	"strings"

	"firebase.google.com/go/v4/auth"
	"github.com/gin-gonic/gin"
)

// OptionalAuthMiddleware attempts to verify the Bearer token.
// If valid, sets *bootstrap.User in context. If missing or invalid, continues without user.
func OptionalAuthMiddleware(client *auth.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.Request.Header.Get("Authorization")
		if header == "" {
			c.Next()
			return
		}

		parts := strings.Split(header, "Bearer ")
		if len(parts) != 2 {
			c.Next()
			return
		}

		tokenID := parts[1]
		token, err := client.VerifyIDToken(context.Background(), tokenID)
		if err != nil {
			slog.Debug("Optional auth: invalid token", "error", err)
			c.Next()
			return
		}

		email, ok := token.Claims["email"].(string)
		if !ok {
			c.Next()
			return
		}

		username, err := client.GetUser(context.Background(), token.UID)
		if err != nil {
			slog.Debug("Optional auth: error getting user", "error", err)
			c.Next()
			return
		}

		user := &bootstrap.User{
			UserID:   token.UID,
			Username: username.DisplayName,
			Email:    email,
		}
		c.Set("user", user)
		c.Next()
	}
}
```

**Step 2: Verify it compiles**

```bash
go build ./api/middleware/...
```

Expected: Clean build.

**Step 3: Commit**

```bash
git add api/middleware/optional_auth.go
git commit -m "feat(middleware): add optional authentication middleware"
```

---

### Task 8: Route Handlers with Tests (TDD)

Implement HTTP handlers and test them with a service stub.

**Files:**
- Create: `api/user_suggestions/route.go`
- Create: `api/user_suggestions/route_test.go`

**Step 1: Create the service stub and test helpers in route_test.go**

```go
package user_suggestions

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"miltechserver/api/response"
	"miltechserver/bootstrap"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

type serviceStub struct {
	suggestions    []SuggestionResponse
	suggestionsErr error
	created        *SuggestionResponse
	createErr      error
	updated        *SuggestionResponse
	updateErr      error
	deleteErr      error
	voteErr        error
	removeVoteErr  error
}

func (s *serviceStub) GetAllSuggestions(currentUser *bootstrap.User) ([]SuggestionResponse, error) {
	return s.suggestions, s.suggestionsErr
}

func (s *serviceStub) CreateSuggestion(user *bootstrap.User, title, description string) (*SuggestionResponse, error) {
	return s.created, s.createErr
}

func (s *serviceStub) UpdateSuggestion(user *bootstrap.User, suggestionID, title, description string) (*SuggestionResponse, error) {
	return s.updated, s.updateErr
}

func (s *serviceStub) DeleteSuggestion(user *bootstrap.User, suggestionID string) error {
	return s.deleteErr
}

func (s *serviceStub) Vote(user *bootstrap.User, suggestionID string, direction int16) error {
	return s.voteErr
}

func (s *serviceStub) RemoveVote(user *bootstrap.User, suggestionID string) error {
	return s.removeVoteErr
}

func setupRouter(svc Service) *gin.Engine {
	router := gin.New()
	publicGroup := router.Group("/api/v1")
	authGroup := router.Group("/api/v1/auth")
	// Simulate auth middleware setting user in context
	authGroup.Use(func(c *gin.Context) {
		user := &bootstrap.User{UserID: "user-1", Username: "testuser", Email: "test@test.com"}
		c.Set("user", user)
		c.Next()
	})
	registerHandlers(publicGroup, authGroup, nil, svc)
	return router
}

func performRequest(router *gin.Engine, method, path string, body interface{}) *httptest.ResponseRecorder {
	var reqBody *bytes.Buffer
	if body != nil {
		jsonBytes, _ := json.Marshal(body)
		reqBody = bytes.NewBuffer(jsonBytes)
	} else {
		reqBody = &bytes.Buffer{}
	}

	req, _ := http.NewRequest(method, path, reqBody)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}
```

**Step 2: Write failing tests for the list and create handlers**

```go
func TestListSuggestions_200(t *testing.T) {
	svc := &serviceStub{
		suggestions: []SuggestionResponse{
			{ID: "abc-123", Title: "Feature A", Score: 5},
		},
	}
	router := setupRouter(svc)

	w := performRequest(router, "GET", "/api/v1/suggestions", nil)
	require.Equal(t, http.StatusOK, w.Code)

	var resp response.StandardResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.Status)
}

func TestCreateSuggestion_201(t *testing.T) {
	svc := &serviceStub{
		created: &SuggestionResponse{
			ID:    "abc-123",
			Title: "New Feature",
		},
	}
	router := setupRouter(svc)

	body := CreateSuggestionRequest{Title: "New Feature", Description: "A great feature"}
	w := performRequest(router, "POST", "/api/v1/auth/suggestions", body)
	require.Equal(t, http.StatusCreated, w.Code)
}

func TestCreateSuggestion_400_Validation(t *testing.T) {
	svc := &serviceStub{createErr: ErrInvalidTitle}
	router := setupRouter(svc)

	body := CreateSuggestionRequest{Title: "", Description: "Desc"}
	w := performRequest(router, "POST", "/api/v1/auth/suggestions", body)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateSuggestion_200(t *testing.T) {
	svc := &serviceStub{
		updated: &SuggestionResponse{ID: "abc-123", Title: "Updated"},
	}
	router := setupRouter(svc)

	body := UpdateSuggestionRequest{Title: "Updated", Description: "Updated desc"}
	w := performRequest(router, "PUT", "/api/v1/auth/suggestions/abc-123", body)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestUpdateSuggestion_403(t *testing.T) {
	svc := &serviceStub{updateErr: ErrForbidden}
	router := setupRouter(svc)

	body := UpdateSuggestionRequest{Title: "Updated", Description: "Updated desc"}
	w := performRequest(router, "PUT", "/api/v1/auth/suggestions/abc-123", body)
	require.Equal(t, http.StatusForbidden, w.Code)
}

func TestUpdateSuggestion_404(t *testing.T) {
	svc := &serviceStub{updateErr: ErrSuggestionNotFound}
	router := setupRouter(svc)

	body := UpdateSuggestionRequest{Title: "Updated", Description: "Updated desc"}
	w := performRequest(router, "PUT", "/api/v1/auth/suggestions/abc-123", body)
	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestDeleteSuggestion_200(t *testing.T) {
	svc := &serviceStub{}
	router := setupRouter(svc)

	w := performRequest(router, "DELETE", "/api/v1/auth/suggestions/abc-123", nil)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestDeleteSuggestion_403(t *testing.T) {
	svc := &serviceStub{deleteErr: ErrForbidden}
	router := setupRouter(svc)

	w := performRequest(router, "DELETE", "/api/v1/auth/suggestions/abc-123", nil)
	require.Equal(t, http.StatusForbidden, w.Code)
}

func TestVote_200(t *testing.T) {
	svc := &serviceStub{}
	router := setupRouter(svc)

	body := VoteRequest{Direction: 1}
	w := performRequest(router, "POST", "/api/v1/auth/suggestions/abc-123/vote", body)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestVote_400_InvalidDirection(t *testing.T) {
	svc := &serviceStub{voteErr: ErrInvalidDirection}
	router := setupRouter(svc)

	body := VoteRequest{Direction: 0}
	w := performRequest(router, "POST", "/api/v1/auth/suggestions/abc-123/vote", body)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRemoveVote_200(t *testing.T) {
	svc := &serviceStub{}
	router := setupRouter(svc)

	w := performRequest(router, "DELETE", "/api/v1/auth/suggestions/abc-123/vote", nil)
	require.Equal(t, http.StatusOK, w.Code)
}
```

**Step 3: Run tests to verify they fail**

```bash
go test ./api/user_suggestions/... -v -run "TestList|TestCreate.*_2|TestCreate.*_4|TestUpdate.*_2|TestUpdate.*_4|TestDelete.*_2|TestDelete.*_4|TestVote.*_2|TestVote.*_4|TestRemoveVote"
```

Expected: FAIL — `registerHandlers` not defined.

**Step 4: Implement route.go**

```go
package user_suggestions

import (
	"database/sql"
	"errors"
	"net/http"

	"miltechserver/api/middleware"
	"miltechserver/api/response"
	"miltechserver/bootstrap"

	"firebase.google.com/go/v4/auth"
	"github.com/gin-gonic/gin"
)

type Dependencies struct {
	DB         *sql.DB
	AuthClient *auth.Client
}

type Handler struct {
	service Service
}

func RegisterRoutes(deps Dependencies, publicGroup, authGroup *gin.RouterGroup) {
	repo := NewRepository(deps.DB)
	svc := NewService(repo)
	registerHandlers(publicGroup, authGroup, deps.AuthClient, svc)
}

func registerHandlers(publicGroup, authGroup *gin.RouterGroup, authClient *auth.Client, svc Service) {
	handler := Handler{service: svc}

	// Public route with optional auth
	if authClient != nil {
		publicGroup.GET("/suggestions", middleware.OptionalAuthMiddleware(authClient), handler.listSuggestions)
	} else {
		// For testing without a real auth client
		publicGroup.GET("/suggestions", handler.listSuggestions)
	}

	authGroup.POST("/suggestions", handler.createSuggestion)
	authGroup.PUT("/suggestions/:id", handler.updateSuggestion)
	authGroup.DELETE("/suggestions/:id", handler.deleteSuggestion)
	authGroup.POST("/suggestions/:id/vote", handler.vote)
	authGroup.DELETE("/suggestions/:id/vote", handler.removeVote)
}

func (h *Handler) listSuggestions(c *gin.Context) {
	var currentUser *bootstrap.User
	if user, exists := c.Get("user"); exists {
		if u, ok := user.(*bootstrap.User); ok {
			currentUser = u
		}
	}

	suggestions, err := h.service.GetAllSuggestions(currentUser)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
		return
	}

	c.JSON(http.StatusOK, response.StandardResponse{
		Status:  http.StatusOK,
		Message: "Suggestions retrieved",
		Data:    suggestions,
	})
}

func (h *Handler) createSuggestion(c *gin.Context) {
	currentUser, ok := getUser(c)
	if !ok {
		return
	}

	var req CreateSuggestionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request body"})
		return
	}

	suggestion, err := h.service.CreateSuggestion(currentUser, req.Title, req.Description)
	if err != nil {
		if respondError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
		return
	}

	c.JSON(http.StatusCreated, response.StandardResponse{
		Status:  http.StatusCreated,
		Message: "Suggestion created",
		Data:    suggestion,
	})
}

func (h *Handler) updateSuggestion(c *gin.Context) {
	currentUser, ok := getUser(c)
	if !ok {
		return
	}

	suggestionID := c.Param("id")

	var req UpdateSuggestionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request body"})
		return
	}

	suggestion, err := h.service.UpdateSuggestion(currentUser, suggestionID, req.Title, req.Description)
	if err != nil {
		if respondError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
		return
	}

	c.JSON(http.StatusOK, response.StandardResponse{
		Status:  http.StatusOK,
		Message: "Suggestion updated",
		Data:    suggestion,
	})
}

func (h *Handler) deleteSuggestion(c *gin.Context) {
	currentUser, ok := getUser(c)
	if !ok {
		return
	}

	suggestionID := c.Param("id")

	err := h.service.DeleteSuggestion(currentUser, suggestionID)
	if err != nil {
		if respondError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
		return
	}

	c.JSON(http.StatusOK, response.StandardResponse{
		Status:  http.StatusOK,
		Message: "Suggestion deleted",
	})
}

func (h *Handler) vote(c *gin.Context) {
	currentUser, ok := getUser(c)
	if !ok {
		return
	}

	suggestionID := c.Param("id")

	var req VoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request body"})
		return
	}

	err := h.service.Vote(currentUser, suggestionID, req.Direction)
	if err != nil {
		if respondError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
		return
	}

	c.JSON(http.StatusOK, response.StandardResponse{
		Status:  http.StatusOK,
		Message: "Vote recorded",
	})
}

func (h *Handler) removeVote(c *gin.Context) {
	currentUser, ok := getUser(c)
	if !ok {
		return
	}

	suggestionID := c.Param("id")

	err := h.service.RemoveVote(currentUser, suggestionID)
	if err != nil {
		if respondError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
		return
	}

	c.JSON(http.StatusOK, response.StandardResponse{
		Status:  http.StatusOK,
		Message: "Vote removed",
	})
}

func getUser(c *gin.Context) (*bootstrap.User, bool) {
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
		return nil, false
	}

	currentUser, ok := user.(*bootstrap.User)
	if !ok || currentUser == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
		return nil, false
	}

	return currentUser, true
}

type errorMapping struct {
	target  error
	status  int
	message string
}

var errorMappings = []errorMapping{
	{target: ErrSuggestionNotFound, status: http.StatusNotFound, message: "suggestion not found"},
	{target: ErrInvalidTitle, status: http.StatusBadRequest, message: "invalid title"},
	{target: ErrInvalidDescription, status: http.StatusBadRequest, message: "invalid description"},
	{target: ErrInvalidDirection, status: http.StatusBadRequest, message: "invalid vote direction"},
	{target: ErrInvalidID, status: http.StatusBadRequest, message: "invalid suggestion ID"},
	{target: ErrUnauthorized, status: http.StatusUnauthorized, message: "unauthorized"},
	{target: ErrForbidden, status: http.StatusForbidden, message: "forbidden"},
}

func respondError(c *gin.Context, err error) bool {
	for _, em := range errorMappings {
		if errors.Is(err, em.target) {
			c.JSON(em.status, gin.H{"message": em.message})
			return true
		}
	}
	return false
}
```

**Step 5: Run all tests**

```bash
go test ./api/user_suggestions/... -v
```

Expected: All PASS.

**Step 6: Commit**

```bash
git add api/user_suggestions/route.go api/user_suggestions/route_test.go
git commit -m "feat(user-suggestions): implement route handlers with full test coverage"
```

---

### Task 9: Repository Implementation

Implement the database queries. This uses raw SQL for the complex `GetAllWithScores` query and Jet for the simpler CRUD operations.

**Files:**
- Create: `api/user_suggestions/repository_impl.go`

**Step 1: Create repository_impl.go**

```go
package user_suggestions

import (
	"database/sql"
	"fmt"
	"time"

	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/.gen/miltech_ng/public/table"

	. "github.com/go-jet/jet/v2/postgres"
	"github.com/go-jet/jet/v2/qrm"
	"github.com/google/uuid"
)

type RepositoryImpl struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *RepositoryImpl {
	return &RepositoryImpl{db: db}
}

func (r *RepositoryImpl) GetAllWithScores(voterID string) ([]SuggestionWithScore, error) {
	rawSQL := `
		SELECT
			s.id, s.user_id, s.title, s.description, s.status,
			s.created_at, s.updated_at,
			u.username,
			COALESCE(SUM(v.direction)::INT, 0) AS score,
			uv.direction AS my_vote
		FROM user_suggestions s
		LEFT JOIN users u ON s.user_id = u.uid
		LEFT JOIN user_suggestion_votes v ON s.id = v.suggestion_id
		LEFT JOIN user_suggestion_votes uv ON s.id = uv.suggestion_id AND uv.voter_id = $1
		GROUP BY s.id, s.user_id, s.title, s.description, s.status,
		         s.created_at, s.updated_at, u.username, uv.direction
		ORDER BY s.created_at DESC
	`

	rows, err := r.db.Query(rawSQL, voterID)
	if err != nil {
		return nil, fmt.Errorf("failed to get suggestions: %w", err)
	}
	defer rows.Close()

	var results []SuggestionWithScore
	for rows.Next() {
		var (
			id          uuid.UUID
			userID      string
			title       string
			description string
			status      string
			createdAt   time.Time
			updatedAt   sql.NullTime
			username    sql.NullString
			score       int
			myVote      sql.NullInt16
		)

		if scanErr := rows.Scan(
			&id, &userID, &title, &description, &status,
			&createdAt, &updatedAt,
			&username,
			&score,
			&myVote,
		); scanErr != nil {
			return nil, fmt.Errorf("failed to scan suggestion row: %w", scanErr)
		}

		sug := SuggestionWithScore{
			ID:          id,
			UserID:      userID,
			Title:       title,
			Description: description,
			Status:      status,
			CreatedAt:   createdAt,
			Score:       score,
		}

		if updatedAt.Valid {
			sug.UpdatedAt = &updatedAt.Time
		}
		if username.Valid {
			sug.Username = &username.String
		}
		if myVote.Valid {
			sug.MyVote = &myVote.Int16
		}

		results = append(results, sug)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating suggestion rows: %w", err)
	}

	return results, nil
}

func (r *RepositoryImpl) GetByID(id uuid.UUID) (*model.UserSuggestions, error) {
	stmt := SELECT(
		table.UserSuggestions.AllColumns,
	).FROM(
		table.UserSuggestions,
	).WHERE(
		table.UserSuggestions.ID.EQ(UUID(id)),
	)

	var suggestion model.UserSuggestions
	err := stmt.Query(r.db, &suggestion)
	if err != nil {
		if err == qrm.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get suggestion: %w", err)
	}

	return &suggestion, nil
}

func (r *RepositoryImpl) Create(suggestion model.UserSuggestions) (*model.UserSuggestions, error) {
	stmt := table.UserSuggestions.INSERT(
		table.UserSuggestions.UserID,
		table.UserSuggestions.Title,
		table.UserSuggestions.Description,
		table.UserSuggestions.Status,
	).VALUES(
		suggestion.UserID,
		suggestion.Title,
		suggestion.Description,
		suggestion.Status,
	).RETURNING(
		table.UserSuggestions.AllColumns,
	)

	var created model.UserSuggestions
	err := stmt.Query(r.db, &created)
	if err != nil {
		return nil, fmt.Errorf("failed to create suggestion: %w", err)
	}

	return &created, nil
}

func (r *RepositoryImpl) Update(id uuid.UUID, title, description string) (*model.UserSuggestions, error) {
	now := time.Now()

	stmt := table.UserSuggestions.UPDATE().
		SET(
			table.UserSuggestions.Title.SET(String(title)),
			table.UserSuggestions.Description.SET(String(description)),
			table.UserSuggestions.UpdatedAt.SET(TimestampzT(now)),
		).
		WHERE(table.UserSuggestions.ID.EQ(UUID(id))).
		RETURNING(table.UserSuggestions.AllColumns)

	var updated model.UserSuggestions
	err := stmt.Query(r.db, &updated)
	if err != nil {
		return nil, fmt.Errorf("failed to update suggestion: %w", err)
	}

	return &updated, nil
}

func (r *RepositoryImpl) Delete(id uuid.UUID) error {
	stmt := table.UserSuggestions.DELETE().
		WHERE(table.UserSuggestions.ID.EQ(UUID(id)))

	_, err := stmt.Exec(r.db)
	if err != nil {
		return fmt.Errorf("failed to delete suggestion: %w", err)
	}

	return nil
}

func (r *RepositoryImpl) UpsertVote(suggestionID uuid.UUID, voterID string, direction int16) error {
	rawSQL := `
		INSERT INTO user_suggestion_votes (suggestion_id, voter_id, direction)
		VALUES ($1, $2, $3)
		ON CONFLICT (suggestion_id, voter_id)
		DO UPDATE SET direction = EXCLUDED.direction, created_at = now()
	`

	_, err := r.db.Exec(rawSQL, suggestionID, voterID, direction)
	if err != nil {
		return fmt.Errorf("failed to upsert vote: %w", err)
	}

	return nil
}

func (r *RepositoryImpl) DeleteVote(suggestionID uuid.UUID, voterID string) error {
	rawSQL := `
		DELETE FROM user_suggestion_votes
		WHERE suggestion_id = $1 AND voter_id = $2
	`

	_, err := r.db.Exec(rawSQL, suggestionID, voterID)
	if err != nil {
		return fmt.Errorf("failed to delete vote: %w", err)
	}

	return nil
}
```

Note: The Jet `table.UserSuggestions` references depend on Jet generation having been run in Task 1. The `TimestampzT` function is the Jet helper for `TIMESTAMPTZ` values — verify the exact function name matches your Jet version (might be `TimestampzT` or `DateTimeT`; check existing code in `item_comments/repository_impl.go` which uses `TimestampT`).

**Step 2: Verify it compiles**

```bash
go build ./api/user_suggestions/...
```

Expected: Clean build.

**Step 3: Commit**

```bash
git add api/user_suggestions/repository_impl.go
git commit -m "feat(user-suggestions): implement repository with Jet and raw SQL queries"
```

---

### Task 10: Wire Into Main Router

Register the new domain in the main route setup.

**Files:**
- Modify: `api/route/route.go` (add import and `RegisterRoutes` call)

**Step 1: Add import and registration**

In `api/route/route.go`, add the import:

```go
"miltechserver/api/user_suggestions"
```

In the `Setup` function, add the registration call. The `user_suggestions` package needs both `v1Route` (public, for GET with optional auth) and `authRoutes` (for all write operations). Add it after the `item_comments` registration:

```go
user_suggestions.RegisterRoutes(user_suggestions.Dependencies{
    DB:         db,
    AuthClient: authClient,
}, v1Route, authRoutes)
```

**Step 2: Verify it compiles**

```bash
go build ./...
```

Expected: Clean build.

**Step 3: Commit**

```bash
git add api/route/route.go
git commit -m "feat(user-suggestions): wire user_suggestions into main router"
```

---

### Task 11: Run Full Test Suite

Verify everything passes.

**Step 1: Run all user_suggestions tests**

```bash
go test ./api/user_suggestions/... -v -count=1
```

Expected: All tests PASS.

**Step 2: Run full project build**

```bash
go build ./...
```

Expected: Clean build, no errors.

**Step 3: Run full test suite (if applicable)**

```bash
go test ./... -count=1
```

Expected: All existing tests still pass (no regressions).

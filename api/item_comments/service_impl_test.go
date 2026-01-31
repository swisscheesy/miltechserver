package item_comments

import (
	"testing"

	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/bootstrap"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type captureRepository struct {
	getNiin       string
	commentByID   *model.ItemComments
	commentByIDErr error
	created       *model.ItemComments
	updated       *model.ItemComments
}

func (repo *captureRepository) GetCommentsByNiin(niin string) ([]CommentWithAuthor, error) {
	repo.getNiin = niin
	return []CommentWithAuthor{}, nil
}

func (repo *captureRepository) GetCommentByID(commentID uuid.UUID) (*model.ItemComments, error) {
	return repo.commentByID, repo.commentByIDErr
}

func (repo *captureRepository) CreateComment(comment model.ItemComments) (*model.ItemComments, error) {
	repo.created = &comment
	return &comment, nil
}

func (repo *captureRepository) UpdateCommentText(commentID uuid.UUID, text string) (*model.ItemComments, error) {
	comment := model.ItemComments{ID: commentID, Text: text}
	repo.updated = &comment
	return &comment, nil
}

func (repo *captureRepository) FlagComment(flag model.ItemCommentFlags) error {
	return nil
}

func TestGetCommentsByNiinNormalizes(t *testing.T) {
	repo := &captureRepository{}
	svc := NewService(repo)

	_, err := svc.GetCommentsByNiin(" 123456789 ")
	require.NoError(t, err)
	require.Equal(t, "123456789", repo.getNiin)
}

func TestCreateCommentValidatesText(t *testing.T) {
	repo := &captureRepository{}
	svc := NewService(repo)
	user := &bootstrap.User{UserID: "user-1", Username: "test"}

	_, err := svc.CreateComment(user, "123456789", "", nil)
	require.ErrorIs(t, err, ErrInvalidText)
}

func TestCreateCommentValidatesParent(t *testing.T) {
	repo := &captureRepository{}
	svc := NewService(repo)
	user := &bootstrap.User{UserID: "user-1", Username: "test"}

	parentID := "not-a-uuid"
	_, err := svc.CreateComment(user, "123456789", "hi", &parentID)
	require.ErrorIs(t, err, ErrInvalidParent)
}

func TestUpdateCommentForbidden(t *testing.T) {
	commentID := uuid.New()
	repo := &captureRepository{
		commentByID: &model.ItemComments{
			ID:          commentID,
			CommentNiin: "123456789",
			AuthorID:    "user-2",
		},
	}
	svc := NewService(repo)
	user := &bootstrap.User{UserID: "user-1", Username: "test"}

	_, err := svc.UpdateComment(user, "123456789", commentID.String(), "text")
	require.ErrorIs(t, err, ErrForbidden)
}

func TestUpdateCommentInvalidText(t *testing.T) {
	commentID := uuid.New()
	repo := &captureRepository{
		commentByID: &model.ItemComments{
			ID:          commentID,
			CommentNiin: "123456789",
			AuthorID:    "user-1",
		},
	}
	svc := NewService(repo)
	user := &bootstrap.User{UserID: "user-1", Username: "test"}

	_, err := svc.UpdateComment(user, "123456789", commentID.String(), "")
	require.ErrorIs(t, err, ErrInvalidText)
}

func TestUpdateCommentInvalidID(t *testing.T) {
	repo := &captureRepository{}
	svc := NewService(repo)
	user := &bootstrap.User{UserID: "user-1", Username: "test"}

	_, err := svc.UpdateComment(user, "123456789", "not-a-uuid", "text")
	require.ErrorIs(t, err, ErrCommentNotFound)
}

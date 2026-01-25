package service

import (
	"errors"
	"strings"

	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/repository"
	"miltechserver/api/response"
	"miltechserver/bootstrap"

	"github.com/google/uuid"
)

const deletedCommentText = "Deleted by user"

var (
	ErrInvalidNiin     = errors.New("invalid NIIN")
	ErrInvalidText     = errors.New("invalid comment text")
	ErrCommentNotFound = errors.New("comment not found")
	ErrUnauthorized    = errors.New("unauthorized user")
	ErrForbidden       = errors.New("user not authorized")
	ErrInvalidParent   = errors.New("invalid parent comment")
)

type ItemCommentsServiceImpl struct {
	repo repository.ItemCommentsRepository
}

func NewItemCommentsServiceImpl(repo repository.ItemCommentsRepository) *ItemCommentsServiceImpl {
	return &ItemCommentsServiceImpl{repo: repo}
}

func (service *ItemCommentsServiceImpl) GetCommentsByNiin(niin string) ([]response.ItemCommentResponse, error) {
	normalized, err := validateNiin(niin)
	if err != nil {
		return nil, err
	}

	comments, err := service.repo.GetCommentsByNiin(normalized)
	if err != nil {
		return nil, err
	}

	result := make([]response.ItemCommentResponse, 0, len(comments))
	for _, comment := range comments {
		result = append(result, mapCommentWithAuthor(comment))
	}

	return result, nil
}

func (service *ItemCommentsServiceImpl) CreateComment(user *bootstrap.User, niin string, text string, parentID *string) (*response.ItemCommentResponse, error) {
	if user == nil {
		return nil, ErrUnauthorized
	}

	normalized, err := validateNiin(niin)
	if err != nil {
		return nil, err
	}

	if len(text) == 0 || len(text) > 255 {
		return nil, ErrInvalidText
	}

	var parentUUID *uuid.UUID
	if parentID != nil {
		parsed, parseErr := uuid.Parse(*parentID)
		if parseErr != nil {
			return nil, ErrInvalidParent
		}

		parent, parentErr := service.repo.GetCommentByID(parsed)
		if parentErr != nil {
			return nil, parentErr
		}
		if parent == nil || parent.CommentNiin != normalized {
			return nil, ErrInvalidParent
		}
		parentUUID = &parsed
	}

	comment := model.ItemComments{
		CommentNiin: normalized,
		AuthorID:    user.UserID,
		Text:        text,
		ParentID:    parentUUID,
	}

	created, err := service.repo.CreateComment(comment)
	if err != nil {
		return nil, err
	}

	resp := mapCommentToResponse(*created, user.Username)
	return &resp, nil
}

func (service *ItemCommentsServiceImpl) UpdateComment(user *bootstrap.User, niin string, commentID string, text string) (*response.ItemCommentResponse, error) {
	if user == nil {
		return nil, ErrUnauthorized
	}

	normalized, err := validateNiin(niin)
	if err != nil {
		return nil, err
	}

	if len(text) == 0 || len(text) > 255 {
		return nil, ErrInvalidText
	}

	commentUUID, err := uuid.Parse(commentID)
	if err != nil {
		return nil, ErrCommentNotFound
	}

	existing, err := service.repo.GetCommentByID(commentUUID)
	if err != nil {
		return nil, err
	}
	if existing == nil || existing.CommentNiin != normalized {
		return nil, ErrCommentNotFound
	}
	if existing.AuthorID != user.UserID {
		return nil, ErrForbidden
	}

	updated, err := service.repo.UpdateCommentText(commentUUID, text)
	if err != nil {
		return nil, err
	}

	resp := mapCommentToResponse(*updated, user.Username)
	return &resp, nil
}

func (service *ItemCommentsServiceImpl) DeleteComment(user *bootstrap.User, niin string, commentID string) (*response.ItemCommentResponse, error) {
	if user == nil {
		return nil, ErrUnauthorized
	}

	normalized, err := validateNiin(niin)
	if err != nil {
		return nil, err
	}

	commentUUID, err := uuid.Parse(commentID)
	if err != nil {
		return nil, ErrCommentNotFound
	}

	existing, err := service.repo.GetCommentByID(commentUUID)
	if err != nil {
		return nil, err
	}
	if existing == nil || existing.CommentNiin != normalized {
		return nil, ErrCommentNotFound
	}
	if existing.AuthorID != user.UserID {
		return nil, ErrForbidden
	}

	updated, err := service.repo.UpdateCommentText(commentUUID, deletedCommentText)
	if err != nil {
		return nil, err
	}

	resp := mapCommentToResponse(*updated, user.Username)
	return &resp, nil
}

func (service *ItemCommentsServiceImpl) FlagComment(user *bootstrap.User, niin string, commentID string) error {
	if user == nil {
		return ErrUnauthorized
	}

	normalized, err := validateNiin(niin)
	if err != nil {
		return err
	}

	commentUUID, err := uuid.Parse(commentID)
	if err != nil {
		return ErrCommentNotFound
	}

	existing, err := service.repo.GetCommentByID(commentUUID)
	if err != nil {
		return err
	}
	if existing == nil || existing.CommentNiin != normalized {
		return ErrCommentNotFound
	}

	flag := model.ItemCommentFlags{
		CommentID: commentUUID,
		FlaggerID: user.UserID,
	}

	return service.repo.FlagComment(flag)
}

func validateNiin(niin string) (string, error) {
	trimmed := strings.TrimSpace(niin)
	if len(trimmed) != 9 {
		return "", ErrInvalidNiin
	}
	for _, char := range trimmed {
		if char < '0' || char > '9' {
			return "", ErrInvalidNiin
		}
	}
	return trimmed, nil
}

func mapCommentWithAuthor(comment repository.ItemCommentWithAuthor) response.ItemCommentResponse {
	authorName := "Unknown"
	if comment.AuthorDisplayName != nil {
		authorName = *comment.AuthorDisplayName
	}

	return mapCommentToResponse(comment.ItemComments, authorName)
}

func mapCommentToResponse(comment model.ItemComments, authorName string) response.ItemCommentResponse {
	var parentID *string
	if comment.ParentID != nil {
		value := comment.ParentID.String()
		parentID = &value
	}

	return response.ItemCommentResponse{
		ID:                comment.ID.String(),
		CommentNiin:       comment.CommentNiin,
		AuthorID:          comment.AuthorID,
		AuthorDisplayName: authorName,
		Text:              comment.Text,
		ParentID:          parentID,
		CreatedAt:         comment.CreatedAt,
	}
}

package shared

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type PreconditionMode string

const (
	PreconditionMatch  PreconditionMode = "match"
	PreconditionCreate PreconditionMode = "create"
)

type Precondition struct {
	Mode PreconditionMode
	ETag string
}

func (precondition Precondition) Matches(currentETag string) bool {
	return precondition.Mode == PreconditionMatch && precondition.ETag == currentETag
}

func MakeChecklistETag(id uuid.UUID, version int64) string {
	return makeETag("checklist", id.String(), version)
}

func MakeSubscriptionETag(checklistID uuid.UUID, version int64) string {
	return makeETag("subscription", checklistID.String(), version)
}

func ParseExistingPrecondition(header string) (Precondition, error) {
	etag := strings.TrimSpace(header)
	if etag == "" {
		return Precondition{}, NewPreconditionRequired("If-Match header is required", nil)
	}
	if !isStrongETag(etag) {
		return Precondition{}, NewInvalidPrecondition("If-Match must contain exactly one strong ETag", nil)
	}
	return Precondition{Mode: PreconditionMatch, ETag: etag}, nil
}

func ParseCreatePrecondition(header string) (Precondition, error) {
	value := strings.TrimSpace(header)
	if value == "" {
		return Precondition{}, NewPreconditionRequired("If-None-Match header is required", nil)
	}
	if value != "*" {
		return Precondition{}, NewInvalidPrecondition("If-None-Match must be *", nil)
	}
	return Precondition{Mode: PreconditionCreate}, nil
}

func DecodeStrictJSON(context *gin.Context, destination any, maxBytes int64) *APIError {
	if maxBytes <= 0 {
		return NewInternalError("invalid JSON body limit", nil)
	}
	if strings.TrimSpace(context.GetHeader("Content-Encoding")) != "" {
		return NewInvalidRequest("request body must not be compressed", nil)
	}

	mediaType, _, err := mime.ParseMediaType(context.GetHeader("Content-Type"))
	if err != nil || mediaType != "application/json" {
		return NewInvalidRequest("request content type must be application/json", nil)
	}

	body := http.MaxBytesReader(context.Writer, context.Request.Body, maxBytes)
	defer body.Close()
	payload, err := io.ReadAll(body)
	if err != nil {
		var maxBytesError *http.MaxBytesError
		if errors.As(err, &maxBytesError) {
			return NewContentTooLarge("request body exceeds the configured limit", nil)
		}
		return NewInvalidRequest("unable to read request body", nil)
	}
	if !utf8.Valid(payload) {
		return NewInvalidRequest("request body must contain valid UTF-8", nil)
	}

	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return NewInvalidRequest("invalid JSON request body", nil)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return NewInvalidRequest("request body must contain exactly one JSON value", nil)
	}
	return nil
}

func WriteAPIError(context *gin.Context, apiError *APIError) {
	if apiError == nil {
		apiError = NewInternalError("unexpected server failure", nil)
	}
	context.JSON(apiError.Status, apiErrorResponse{
		Status:  apiError.Status,
		Message: apiError.Message,
		Data:    nil,
		Error: apiErrorResponseDetails{
			Code:    apiError.Code,
			Details: apiError.Details,
		},
	})
}

type apiErrorResponse struct {
	Status  int                     `json:"status"`
	Message string                  `json:"message"`
	Data    any                     `json:"data"`
	Error   apiErrorResponseDetails `json:"error"`
}

type apiErrorResponseDetails struct {
	Code    string         `json:"code"`
	Details map[string]any `json:"details,omitempty"`
}

func makeETag(kind, identity string, version int64) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s\x00%s\x00%d", kind, identity, version)))
	return `"` + base64.RawURLEncoding.EncodeToString(sum[:]) + `"`
}

func isStrongETag(value string) bool {
	if len(value) < 2 || value[0] != '"' || value[len(value)-1] != '"' {
		return false
	}
	for _, character := range value[1 : len(value)-1] {
		if character == '"' || character <= 0x20 || character == 0x7f {
			return false
		}
	}
	return true
}

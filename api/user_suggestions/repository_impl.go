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
		WHERE s.show IS TRUE
		GROUP BY s.id, s.user_id, s.title, s.description, s.status,
		         s.created_at, s.updated_at, u.username, uv.direction
		ORDER BY score DESC, s.created_at DESC
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

func (r *RepositoryImpl) GetVote(suggestionID uuid.UUID, voterID string) (*int16, error) {
	rawSQL := `
		SELECT direction FROM user_suggestion_votes
		WHERE suggestion_id = $1 AND voter_id = $2
	`

	var direction int16
	err := r.db.QueryRow(rawSQL, suggestionID, voterID).Scan(&direction)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get vote: %w", err)
	}

	return &direction, nil
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

package pmcs_sbs_progress_test

import (
	"database/sql"
	"testing"
	"time"

	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/bootstrap"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func testUser(id string) *bootstrap.User {
	return &bootstrap.User{UserID: id, Username: id, Email: id + "@example.com"}
}

func ensureUser(t *testing.T, db *sql.DB, user *bootstrap.User) {
	t.Helper()

	now := time.Now().UTC()
	_, err := db.Exec(
		`INSERT INTO users (uid, email, username, created_at, is_enabled)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (uid) DO NOTHING`,
		user.UserID,
		user.Email,
		user.Username,
		now,
		true,
	)
	require.NoError(t, err)
}

func clearPmcsSbsTables(t *testing.T, db *sql.DB) {
	t.Helper()
	_, err := db.Exec(`TRUNCATE TABLE pmcs_sbs_faults, pmcs_sbs_completions, pmcs_sbs_equipment RESTART IDENTITY CASCADE`)
	require.NoError(t, err)
}

func sampleEquipment(user *bootstrap.User) model.PmcsSbsEquipment {
	now := time.Now().UTC()
	return model.PmcsSbsEquipment{
		ID:              uuid.New(),
		UserUID:         user.UserID,
		EquipmentManual: "pmcs_sbs/hmmwv/basic.json",
		Admin:           "A12",
		Serial:          "SER-1",
		Uic:             "UIC",
		CreatedAt:       now,
		UpdatedAt:       now,
	}
}

func sampleCompletion(equipmentID uuid.UUID) model.PmcsSbsCompletions {
	return model.PmcsSbsCompletions{
		EquipmentID: equipmentID,
		SectionID:   "before",
		ItemIndex:   0,
		ItemNo:      "1",
		StepID:      "1-a",
		IsComplete:  true,
		UpdatedAt:   time.Now().UTC(),
	}
}

func sampleFault(equipmentID uuid.UUID) model.PmcsSbsFaults {
	now := time.Now().UTC()
	return model.PmcsSbsFaults{
		EquipmentID:      equipmentID,
		SectionID:        "before",
		ItemIndex:        0,
		ItemNo:           "1",
		Status:           "x",
		FaultText:        "leak",
		CorrectiveAction: "",
		CreatedAt:        now,
		UpdatedAt:        now,
	}
}

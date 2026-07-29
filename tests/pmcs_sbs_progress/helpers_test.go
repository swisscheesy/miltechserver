package pmcs_sbs_progress_test

import (
	"database/sql"
	"fmt"
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
	_, err := db.Exec(
		`TRUNCATE TABLE
			pmcs_sbs_faults,
			pmcs_sbs_inspection_comments,
			pmcs_sbs_inspections,
			shop_vehicle_notification_changes,
			shop_vehicle_notifications,
			shop_vehicle,
			shop_members,
			shops
		RESTART IDENTITY CASCADE`,
	)
	require.NoError(t, err)
}

func createShopWithMember(t *testing.T, db *sql.DB, user *bootstrap.User, role string) string {
	t.Helper()

	shopID := uuid.New().String()
	memberID := uuid.New().String()
	now := time.Now().UTC()
	_, err := db.Exec(
		`INSERT INTO shops (id, name, details, created_by, created_at, updated_at, admin_only_lists)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		shopID,
		"PMCS Shop",
		"Details",
		user.UserID,
		now,
		now,
		false,
	)
	require.NoError(t, err)

	_, err = db.Exec(
		`INSERT INTO shop_members (id, shop_id, user_id, role, joined_at)
		 VALUES ($1, $2, $3, $4, $5)`,
		memberID,
		shopID,
		user.UserID,
		role,
		now,
	)
	require.NoError(t, err)
	return shopID
}

func addShopMember(t *testing.T, db *sql.DB, shopID string, user *bootstrap.User, role string) {
	t.Helper()

	_, err := db.Exec(
		`INSERT INTO shop_members (id, shop_id, user_id, role, joined_at)
		 VALUES ($1, $2, $3, $4, $5)`,
		uuid.New().String(),
		shopID,
		user.UserID,
		role,
		time.Now().UTC(),
	)
	require.NoError(t, err)
}

func createShopVehicle(t *testing.T, db *sql.DB, shopID string, creator *bootstrap.User, admin string) string {
	t.Helper()

	vehicleID := uuid.New().String()
	now := time.Now().UTC()
	_, err := db.Exec(
		`INSERT INTO shop_vehicle (
			id, creator_id, niin, admin, model, serial, uoc, mileage, hours, comment,
			save_time, last_updated, shop_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`,
		vehicleID,
		creator.UserID,
		"",
		admin,
		"M1152A1",
		fmt.Sprintf("SER-%s", admin),
		"UNK",
		0,
		0,
		"",
		now,
		now,
		shopID,
	)
	require.NoError(t, err)
	return vehicleID
}

func sampleInspection(equipmentID string, performedBy string) model.PmcsSbsInspections {
	performedByCopy := performedBy
	return model.PmcsSbsInspections{
		ID:            uuid.New(),
		EquipmentID:   equipmentID,
		GuideManual:   "pmcs_sbs/hmmwv/file.json",
		PerformedDate: time.Now().UTC(),
		PerformedBy:   &performedByCopy,
	}
}

func sampleFault(pmcsID uuid.UUID) model.PmcsSbsFaults {
	now := time.Now().UTC()
	return model.PmcsSbsFaults{
		PmcsID:           pmcsID,
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

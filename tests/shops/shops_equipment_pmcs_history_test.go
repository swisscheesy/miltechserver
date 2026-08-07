package shops_test

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"miltechserver/api/shops/aggregates"
	"miltechserver/bootstrap"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

func TestGetEquipmentPmcsHistoryRepositoryIncludesGuideAndCustom(t *testing.T) {
	const (
		historyUserID = "history-custom-user"
		customUserID  = "custom-performer"
		hiddenUserID  = "hidden-custom-user"
	)
	clearShopTables(t, testDB)
	t.Cleanup(func() {
		clearShopTables(t, testDB)
		_, err := testDB.Exec(`DELETE FROM users WHERE uid IN ($1, $2, $3)`, historyUserID, customUserID, hiddenUserID)
		require.NoError(t, err)
	})
	ensureUser(t, testDB, historyUserID)
	ensureUser(t, testDB, customUserID)
	ensureUser(t, testDB, hiddenUserID)
	_, err := testDB.Exec(`UPDATE users SET username='Guide Tech' WHERE uid=$1`, historyUserID)
	require.NoError(t, err)
	_, err = testDB.Exec(`UPDATE users SET username='Custom Tech' WHERE uid=$1`, customUserID)
	require.NoError(t, err)
	router := newTestRouter(t)

	shopID := createShop(t, router, historyUserID, "History Shop")
	hiddenShopID := createShop(t, router, hiddenUserID, "Hidden Shop")
	vehicleWithHistoryID := createVehicle(t, router, historyUserID, shopID)
	hiddenVehicleID := createVehicle(t, router, hiddenUserID, hiddenShopID)

	// Create second vehicle in same shop by creating another shop first, then inserting
	shop2ID := createShop(t, router, historyUserID, "History Shop 2")
	vehicleWithoutHistoryID := createVehicle(t, router, historyUserID, shop2ID)

	// Update vehicles with unique admin/serial
	_, err = testDB.Exec(`UPDATE shop_vehicle SET admin='A1', serial='S1' WHERE id=$1`, vehicleWithHistoryID)
	require.NoError(t, err)
	_, err = testDB.Exec(`UPDATE shop_vehicle SET admin='A2', serial='S2' WHERE id=$1`, vehicleWithoutHistoryID)
	require.NoError(t, err)
	_, err = testDB.Exec(`UPDATE shop_vehicle SET admin='A3', serial='S3' WHERE id=$1`, hiddenVehicleID)
	require.NoError(t, err)

	newerTime := time.Date(2026, time.July, 16, 14, 30, 0, 0, time.UTC)
	olderTime := newerTime.Add(-7 * 24 * time.Hour)
	guideInspectionID := createPmcsInspection(t, testDB, vehicleWithHistoryID, "pmcs_sbs/hmmwv/hmmwv_up_armor_pmcs.json", olderTime, historyUserID)
	createPmcsFault(t, testDB, guideInspectionID, "before", 0)
	createPmcsFault(t, testDB, guideInspectionID, "during", 1)
	createPmcsComment(t, testDB, guideInspectionID, historyUserID, "guide comment")
	customInspection := createCustomPmcsHistoryFixture(t, testDB, vehicleWithHistoryID, newerTime, customUserID)
	createPmcsFault(t, testDB, customInspection.ID, "custom", 0)
	createPmcsComment(t, testDB, customInspection.ID, customUserID, "first custom comment")
	createPmcsComment(t, testDB, customInspection.ID, customUserID, "second custom comment")
	hiddenInspection := createCustomPmcsHistoryFixture(t, testDB, hiddenVehicleID, newerTime.Add(time.Hour), hiddenUserID)
	createPmcsFault(t, testDB, hiddenInspection.ID, "hidden", 0)

	countingDB, queryCounter := newShopHistoryQueryCountingDatabase(t)
	repository := aggregates.NewRepository(countingDB)
	equipment, err := repository.GetEquipmentPmcsHistory(context.Background(), &bootstrap.User{UserID: historyUserID})

	require.NoError(t, err)
	require.Len(t, equipment, 2)

	byID := make(map[string]int, len(equipment))
	for i, e := range equipment {
		byID[e.ID] = i
	}
	require.Contains(t, byID, vehicleWithHistoryID)
	require.Contains(t, byID, vehicleWithoutHistoryID)

	withHistory := equipment[byID[vehicleWithHistoryID]]
	require.Equal(t, shopID, withHistory.ShopID)
	require.Len(t, withHistory.HistoricalPmcs, 2)
	require.Equal(t, customInspection.ID, withHistory.HistoricalPmcs[0].ID.String())
	require.Equal(t, "custom", withHistory.HistoricalPmcs[0].SourceType)
	require.Nil(t, withHistory.HistoricalPmcs[0].GuideManual)
	require.Equal(t, customInspection.ChecklistID, withHistory.HistoricalPmcs[0].CustomChecklistID.String())
	require.Equal(t, customInspection.RevisionID, withHistory.HistoricalPmcs[0].CustomRevisionID.String())
	require.Equal(t, customInspection.RevisionNumber, *withHistory.HistoricalPmcs[0].CustomRevisionNumber)
	require.Equal(t, customInspection.ChecklistName, *withHistory.HistoricalPmcs[0].CustomChecklistName)
	require.Equal(t, 1, withHistory.HistoricalPmcs[0].FaultCount)
	require.Equal(t, 2, withHistory.HistoricalPmcs[0].CommentCount)
	require.Equal(t, customUserID, *withHistory.HistoricalPmcs[0].PerformedBy)
	require.Equal(t, "Custom Tech", *withHistory.HistoricalPmcs[0].PerformedByUsername)
	require.Equal(t, guideInspectionID, withHistory.HistoricalPmcs[1].ID.String())
	require.Equal(t, "guide", withHistory.HistoricalPmcs[1].SourceType)
	require.Equal(t, "pmcs_sbs/hmmwv/hmmwv_up_armor_pmcs.json", *withHistory.HistoricalPmcs[1].GuideManual)
	require.Nil(t, withHistory.HistoricalPmcs[1].CustomChecklistID)
	require.Nil(t, withHistory.HistoricalPmcs[1].CustomRevisionID)
	require.Nil(t, withHistory.HistoricalPmcs[1].CustomRevisionNumber)
	require.Nil(t, withHistory.HistoricalPmcs[1].CustomChecklistName)
	require.Equal(t, 2, withHistory.HistoricalPmcs[1].FaultCount)
	require.Equal(t, 1, withHistory.HistoricalPmcs[1].CommentCount)
	require.Equal(t, historyUserID, *withHistory.HistoricalPmcs[1].PerformedBy)
	require.Equal(t, "Guide Tech", *withHistory.HistoricalPmcs[1].PerformedByUsername)

	withoutHistory := equipment[byID[vehicleWithoutHistoryID]]
	require.Equal(t, shop2ID, withoutHistory.ShopID)
	require.Empty(t, withoutHistory.HistoricalPmcs)

	for _, e := range equipment {
		require.NotEqual(t, hiddenShopID, e.ShopID)
	}

	queries := queryCounter.snapshot()
	require.Len(t, queries, 4, "equipment history must stay fixed at four batched queries")
	require.Equal(t, 1, countQueriesContaining(queries, "pmcs_sbs_inspections"))
	require.Equal(t, 1, countQueriesContaining(queries, "pmcs_sbs_faults"))
	require.Equal(t, 1, countQueriesContaining(queries, "pmcs_sbs_inspection_comments"))
}

func TestGetEquipmentPmcsHistoryRepositoryReturnsEmptyForUserWithNoShops(t *testing.T) {
	clearShopTables(t, testDB)
	ensureUser(t, testDB, "lonely-user")

	repository := aggregates.NewRepository(testDB)
	equipment, err := repository.GetEquipmentPmcsHistory(context.Background(), &bootstrap.User{UserID: "lonely-user"})

	require.NoError(t, err)
	require.Empty(t, equipment)
}

func TestGetEquipmentPmcsHistoryRepositoryHonorsCanceledContext(t *testing.T) {
	clearShopTables(t, testDB)
	ensureUser(t, testDB, "history-user")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	repository := aggregates.NewRepository(testDB)
	_, err := repository.GetEquipmentPmcsHistory(ctx, &bootstrap.User{UserID: "history-user"})

	require.Error(t, err)
	require.True(t, errors.Is(err, context.Canceled), err.Error())
}

func TestEquipmentPmcsHistoryEndpoint(t *testing.T) {
	clearShopTables(t, testDB)
	ensureUser(t, testDB, "history-user")
	ensureUser(t, testDB, "other-user")
	router := newTestRouter(t)

	shopID := createShop(t, router, "history-user", "History Shop")
	hiddenShopID := createShop(t, router, "other-user", "Hidden Shop")
	vehicleID := createVehicle(t, router, "history-user", shopID)
	_ = createVehicle(t, router, "other-user", hiddenShopID)
	_, err := testDB.Exec(`UPDATE shop_vehicle SET admin='A1', model='M1152A1', serial='SER-1', niin='000000001' WHERE id=$1`, vehicleID)
	require.NoError(t, err)

	performedDate := time.Date(2026, time.July, 16, 14, 30, 0, 0, time.UTC)
	inspectionID := createPmcsInspection(t, testDB, vehicleID, "pmcs_sbs/hmmwv/hmmwv_up_armor_pmcs.json", performedDate, "history-user")
	createPmcsFault(t, testDB, inspectionID, "before", 0)
	createPmcsComment(t, testDB, inspectionID, "history-user", "a comment")

	resp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/equipment-pmcs-history", nil, "history-user")
	require.Equal(t, http.StatusOK, resp.Code)

	payload := decodeMap(t, decodeStandardResponse(t, resp.Body).Data)
	equipment := payload["equipment"].([]interface{})
	require.Len(t, equipment, 1)

	entry := equipment[0].(map[string]interface{})
	require.Equal(t, vehicleID, entry["id"])
	require.Equal(t, shopID, entry["shop_id"])
	require.Equal(t, "A1", entry["admin"])

	history := entry["historical_pmcs"].([]interface{})
	require.Len(t, history, 1)
	inspection := history[0].(map[string]interface{})
	require.Equal(t, inspectionID, inspection["id"])
	require.Equal(t, "pmcs_sbs/hmmwv/hmmwv_up_armor_pmcs.json", inspection["guide_manual"])
	require.Equal(t, float64(1), inspection["fault_count"])
	require.Equal(t, float64(1), inspection["comment_count"])
	require.Equal(t, "history-user", inspection["performed_by"])
	require.Equal(t, "test-user", inspection["performed_by_username"])
}

func TestGetEquipmentPmcsHistoryRepositoryIncludesPerformedByUsername(t *testing.T) {
	clearShopTables(t, testDB)
	ensureUser(t, testDB, "history-user")
	router := newTestRouter(t)

	shopID := createShop(t, router, "history-user", "History Shop")
	vehicleID := createVehicle(t, router, "history-user", shopID)

	performedDate := time.Date(2026, time.July, 16, 14, 30, 0, 0, time.UTC)
	inspectionID := createPmcsInspection(t, testDB, vehicleID, "pmcs_sbs/hmmwv/hmmwv_up_armor_pmcs.json", performedDate, "history-user")

	repository := aggregates.NewRepository(testDB)
	equipment, err := repository.GetEquipmentPmcsHistory(context.Background(), &bootstrap.User{UserID: "history-user"})

	require.NoError(t, err)
	require.Len(t, equipment, 1)
	require.Len(t, equipment[0].HistoricalPmcs, 1)
	entry := equipment[0].HistoricalPmcs[0]
	require.Equal(t, inspectionID, entry.ID.String())
	require.NotNil(t, entry.PerformedBy)
	require.Equal(t, "history-user", *entry.PerformedBy)
	require.NotNil(t, entry.PerformedByUsername)
	require.Equal(t, "test-user", *entry.PerformedByUsername)
}

func TestEquipmentPmcsHistoryEndpointRequiresAuthentication(t *testing.T) {
	router := newTestRouter(t)
	resp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/equipment-pmcs-history", nil, "")
	require.Equal(t, http.StatusUnauthorized, resp.Code)
}

type customPmcsHistoryFixture struct {
	ID             string
	ChecklistID    string
	RevisionID     string
	RevisionNumber int32
	ChecklistName  string
}

func createCustomPmcsHistoryFixture(t *testing.T, db *sql.DB, equipmentID string, performedDate time.Time, performedBy string) customPmcsHistoryFixture {
	t.Helper()

	fixture := customPmcsHistoryFixture{
		ID:             uuid.NewString(),
		ChecklistID:    uuid.NewString(),
		RevisionID:     uuid.NewString(),
		RevisionNumber: 0,
		ChecklistName:  "Device Checklist",
	}
	_, err := db.Exec(
		`INSERT INTO pmcs_sbs_inspections
		  (id, equipment_id, source_type, guide_manual,
		   custom_checklist_id, custom_revision_id,
		   custom_revision_number, custom_checklist_name,
		   performed_date, performed_by)
		 VALUES ($1, $2, 'custom', NULL, $3, $4, $5, $6, $7, $8)`,
		fixture.ID,
		equipmentID,
		fixture.ChecklistID,
		fixture.RevisionID,
		fixture.RevisionNumber,
		fixture.ChecklistName,
		performedDate,
		performedBy,
	)
	require.NoError(t, err)
	return fixture
}

type shopHistoryQueryCounter struct {
	mutex   sync.Mutex
	queries []string
}

func (counter *shopHistoryQueryCounter) record(query string) {
	counter.mutex.Lock()
	defer counter.mutex.Unlock()

	counter.queries = append(counter.queries, query)
}

func (counter *shopHistoryQueryCounter) snapshot() []string {
	counter.mutex.Lock()
	defer counter.mutex.Unlock()

	queries := make([]string, len(counter.queries))
	copy(queries, counter.queries)
	return queries
}

type shopHistoryQueryCountingConnector struct {
	base    driver.Connector
	counter *shopHistoryQueryCounter
}

func (connector *shopHistoryQueryCountingConnector) Connect(ctx context.Context) (driver.Conn, error) {
	connection, err := connector.base.Connect(ctx)
	if err != nil {
		return nil, err
	}
	return &shopHistoryQueryCountingConnection{Conn: connection, counter: connector.counter}, nil
}

func (connector *shopHistoryQueryCountingConnector) Driver() driver.Driver {
	return connector.base.Driver()
}

type shopHistoryQueryCountingConnection struct {
	driver.Conn
	counter *shopHistoryQueryCounter
}

func (connection *shopHistoryQueryCountingConnection) QueryContext(ctx context.Context, query string, arguments []driver.NamedValue) (driver.Rows, error) {
	queryer, ok := connection.Conn.(driver.QueryerContext)
	if !ok {
		return nil, driver.ErrSkip
	}
	connection.counter.record(query)
	return queryer.QueryContext(ctx, query, arguments)
}

func newShopHistoryQueryCountingDatabase(t *testing.T) (*sql.DB, *shopHistoryQueryCounter) {
	t.Helper()

	databaseURL, err := url.Parse(os.Getenv("TEST_DATABASE_URL"))
	require.NoError(t, err)
	query := databaseURL.Query()
	if query.Get("sslmode") == "" {
		query.Set("sslmode", "disable")
		databaseURL.RawQuery = query.Encode()
	}
	connector, err := pq.NewConnector(databaseURL.String())
	require.NoError(t, err)
	counter := &shopHistoryQueryCounter{}
	database := sql.OpenDB(&shopHistoryQueryCountingConnector{base: connector, counter: counter})
	t.Cleanup(func() { require.NoError(t, database.Close()) })

	var databaseName string
	require.NoError(t, database.QueryRow(`SELECT current_database()`).Scan(&databaseName))
	require.Equal(t, "miltech_ng_test", databaseName)
	counter.mutex.Lock()
	counter.queries = nil
	counter.mutex.Unlock()
	return database, counter
}

func countQueriesContaining(queries []string, fragment string) int {
	count := 0
	for _, query := range queries {
		if strings.Contains(strings.ToLower(query), fragment) {
			count++
		}
	}
	return count
}

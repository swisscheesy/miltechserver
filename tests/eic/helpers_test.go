package eic_test

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"miltechserver/api/eic"
	"miltechserver/api/middleware"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type standardResponse struct {
	Status  int             `json:"status"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func newTestRouter(t *testing.T) *gin.Engine {
	t.Helper()

	return newTestRouterWithDB(t, testDB)
}

func newTestRouterWithDB(t *testing.T, db *sql.DB) *gin.Engine {
	t.Helper()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.ErrorHandler)

	publicGroup := router.Group("/api/v1")
	eic.RegisterRoutes(eic.Dependencies{DB: db}, publicGroup)

	return router
}

func doJSONRequest(t *testing.T, router *gin.Engine, method string, path string) *httptest.ResponseRecorder {
	t.Helper()

	req, err := http.NewRequest(method, path, strings.NewReader(""))
	require.NoError(t, err)

	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func hasRelation(t *testing.T, db *sql.DB, relation string) bool {
	t.Helper()

	var exists bool
	err := db.QueryRow(`SELECT to_regclass($1) IS NOT NULL`, "public."+relation).Scan(&exists)
	require.NoError(t, err)
	return exists
}

func countRows(t *testing.T, db *sql.DB, relation string) int {
	t.Helper()

	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM " + relation).Scan(&count)
	require.NoError(t, err)
	return count
}

func fetchEicSample(t *testing.T, db *sql.DB) (string, string, string, bool) {
	t.Helper()

	if !hasRelation(t, db, "eic") {
		return "", "", "", false
	}

	var niin sql.NullString
	var lin sql.NullString
	var fsc sql.NullString
	err := db.QueryRow("SELECT niin, lin, fsc FROM eic LIMIT 1").Scan(&niin, &lin, &fsc)
	if err == sql.ErrNoRows {
		return "", "", "", false
	}
	require.NoError(t, err)
	if !niin.Valid || !lin.Valid || !fsc.Valid {
		return "", "", "", false
	}
	return niin.String, lin.String, fsc.String, true
}

func countConsolidatedEic(t *testing.T, db *sql.DB, whereClause string, args ...interface{}) int {
	t.Helper()

	query := `
SELECT COUNT(*) FROM (
	SELECT 1
	FROM eic
	` + whereClause + `
	GROUP BY inc, fsc, niin, eic, lin, nomen, model, eicc, ecc, cmdtycd, reported, dahr,
		publvl1, pubno1, pubdate1, pubchg1, pubcgdt1,
		publcl2, pubno2, pubdate2, pubchg2, pubcgdt2,
		publvl3, pubno3, pubdate3, pubchg3, pubcgdt3,
		publvl4, pubno4, pubdate4, pubchg4, pubcgdt4,
		publvl5, pubno5, pubdate5, pubchg5, pubcgdt5,
		publvl6, pubno6, pubdate6, pubchg6, pubcgdt6,
		publvl7, pubno7, pubdate7, pubchg7, pubcgdt7,
		pubremks, eqpmcsa, eqpmcsb, eqpmcsc, eqpmcsd, eqpmcse, eqpmcsf,
		eqpmcsg, eqpmcsh, eqpmcsi, eqpmcsj, eqpmcsk, eqpmcsl,
		wpnrec, sernotrk, orf, aoap, gainloss, usage, urm1, urm2,
		uom1, uom2, uom3, mau1, uom4, mau2,
		warranty, rbm, sos, erc, eslvl, oslin, lcc, nounabb,
		curfmc, prevfmc, bstat1, bstat2, matcat, itemmgr, eos, sorts, status, lst_updt
) AS consolidated_count
`

	var count int
	err := db.QueryRow(query, args...).Scan(&count)
	require.NoError(t, err)
	return count
}

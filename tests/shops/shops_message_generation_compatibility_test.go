package shops_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/require"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"miltechserver/api/shops/messages"
	"miltechserver/bootstrap"
	"miltechserver/internal/jetgen"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestShopMessageNineKeysAfterGeneration(t *testing.T) {
	// TestMain has already checked loopback identity and the disposable marker.
	workspace := t.TempDir()
	output := filepath.Join(workspace, ".gen")
	var missing bool
	require.NoError(t, testDB.QueryRow(`SELECT to_regclass('public.lookup_lin_niin_mat') IS NULL`).Scan(&missing))
	t.Logf("approved fixture catalog: lookup_lin_niin_mat absent=%t (checked tables, views, materialized views)", missing)
	require.NoError(t, jetgen.Generate(testDB, jetgen.Config{Schema: "public", OutputDirectory: output}))
	manifestBytes, err := os.ReadFile(filepath.Join(output, "miltech_ng/public/generation-manifest.json"))
	require.NoError(t, err)
	var manifest jetgen.Manifest
	require.NoError(t, json.Unmarshal(manifestBytes, &manifest))
	require.Equal(t, "miltech_test_shops", manifest.Database)
	source, err := url.Parse(os.Getenv("TEST_DATABASE_URL"))
	require.NoError(t, err)
	require.Equal(t, source.Port(), strconv.Itoa(manifest.Port))
	require.Len(t, manifest.CatalogSHA256, 64)
	t.Logf("generated source database=%s address=%s port=%d schema=%s catalog=%s generator=%s", manifest.Database, manifest.Address, manifest.Port, manifest.Schema, manifest.CatalogSHA256, manifest.GeneratorVersion)
	root, err := filepath.Abs("../..")
	require.NoError(t, err)
	// Compile the actual handwritten DTO declarations against the newly generated
	// model package. Other application models depend on objects absent from the
	// approved fixture, so this is deliberately a focused contract probe.
	sourceFile, err := parser.ParseFile(token.NewFileSet(), filepath.Join(root, "api/response/user_shops_response.go"), nil, 0)
	require.NoError(t, err)
	var declarations bytes.Buffer
	declarations.WriteString("package response\nimport (\"time\";\"miltechserver/.gen/miltech_ng/public/model\")\n")
	for _, declaration := range sourceFile.Decls {
		keep := false
		switch node := declaration.(type) {
		case *ast.GenDecl:
			for _, spec := range node.Specs {
				if named, ok := spec.(*ast.TypeSpec); ok && named.Name.Name == "ShopMessageResponse" {
					keep = true
				}
			}
		case *ast.FuncDecl:
			keep = node.Name.Name == "NewShopMessageResponse"
		}
		if keep {
			require.NoError(t, format.Node(&declarations, token.NewFileSet(), declaration))
			declarations.WriteByte('\n')
		}
	}
	probe := filepath.Join(workspace, "probe")
	require.NoError(t, os.Mkdir(probe, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(probe, "response.go"), declarations.Bytes(), 0644))
	dtoTest, err := os.ReadFile(filepath.Join(root, "api/response/shop_message_response_test.go"))
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(probe, "response_test.go"), dtoTest, 0644))
	for _, name := range []string{"go.mod", "go.sum"} {
		data, err := os.ReadFile(filepath.Join(root, name))
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(filepath.Join(workspace, name), data, 0644))
	}
	references := map[string]string{}
	if manifestPath := os.Getenv("SHOPS_GENERATION_REFERENCE_MANIFEST"); manifestPath != "" {
		data, err := os.ReadFile(manifestPath)
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal(data, &references))
		t.Log("using wrapper hash-only reference manifest; no Git metadata required")
	} else {
		trackedCommand := exec.Command("git", "ls-files", ".gen")
		trackedCommand.Dir = root
		tracked, err := trackedCommand.Output()
		require.NoError(t, err)
		for _, name := range strings.Fields(string(tracked)) {
			if !strings.HasPrefix(filepath.Base(name), "user_pmcs_") {
				continue
			}
			before, err := os.ReadFile(filepath.Join(root, name))
			require.NoError(t, err)
			digest := sha256.Sum256(before)
			references[name] = hex.EncodeToString(digest[:])
		}
	}
	require.NoError(t, validateUserPMCSReferenceHashes(references))
	for name, expected := range references {
		after, err := os.ReadFile(filepath.Join(workspace, name))
		require.NoError(t, err)
		digest := sha256.Sum256(after)
		require.Equal(t, expected, hex.EncodeToString(digest[:]), "tracked generation changed: %s", name)
	}
	t.Logf("tracked user_pmcs generated files compared unchanged: %d", len(references))
	command := exec.Command("go", "test", "./probe", "./.gen/miltech_ng/public/model", "./.gen/miltech_ng/public/table", "./.gen/miltech_ng/public/view", "-count=1")
	command.Dir = workspace
	for _, variable := range os.Environ() {
		if !strings.HasPrefix(variable, "TEST_DATABASE_URL=") && !strings.HasPrefix(variable, "TEST_DATABASE_MARKER=") && !strings.HasPrefix(variable, "TEST_DB_URL=") {
			command.Env = append(command.Env, variable)
		}
	}
	result, err := command.CombinedOutput()
	require.NoError(t, err, "generated consumers: %s", result)
	t.Logf("generated consumers: %s", result)

	clearShopTables(t, testDB)
	ensureUser(t, testDB, "user-1")
	router := newTestRouterWithFlags(t, false, true)
	shopID := createShop(t, router, "user-1", "Generated DTO")
	sent := doJSONRequest(t, router, http.MethodPost, "/api/v1/auth/shops/messages", map[string]interface{}{"shop_id": shopID, "message": "persisted generation message"}, "user-1")
	require.Equal(t, http.StatusCreated, sent.Code, sent.Body.String())
	created := decodeMap(t, decodeStandardResponse(t, sent.Body).Data)
	id := created["id"].(string)
	assertMessage := func(message map[string]interface{}) {
		requireMessageKeys(t, message)
		require.Equal(t, id, message["id"])
		require.Equal(t, shopID, message["shop_id"])
		require.Equal(t, "user-1", message["user_id"])
		require.Equal(t, "persisted generation message", message["message"])
		require.Nil(t, message["parent_id"])
		require.Equal(t, false, message["is_edited"])
		require.Equal(t, created["created_at"], message["created_at"])
		require.Equal(t, created["updated_at"], message["updated_at"])
		require.Equal(t, created["author_username"], message["author_username"])
	}
	assertMessage(created)
	_, err = time.Parse(time.RFC3339Nano, created["created_at"].(string))
	require.NoError(t, err)
	for _, path := range []string{"/api/v1/auth/shops/" + shopID + "/messages", "/api/v1/auth/shops/" + shopID + "/messages/paginated?page=1&limit=10", "/api/v1/auth/shops/" + shopID + "/snapshot?include=messages"} {
		response := doJSONRequest(t, router, http.MethodGet, path, nil, "user-1")
		require.Equal(t, http.StatusOK, response.Code, response.Body.String())
		var payload interface{}
		require.NoError(t, json.Unmarshal(decodeStandardResponse(t, response.Body).Data, &payload))
		var rows []interface{}
		switch value := payload.(type) {
		case []interface{}:
			rows = value
		case map[string]interface{}:
			rows = value["messages"].([]interface{})
		}
		require.Len(t, rows, 1, path)
		assertMessage(rows[0].(map[string]interface{}))
	}
	repo := messages.NewRepository(testDB, nil, nil)
	cursorRows, err := repo.GetShopMessagesByCursor(context.Background(), &bootstrap.User{UserID: "user-1"}, shopID, id, time.Now().Add(time.Hour), true, 10)
	require.NoError(t, err)
	require.Len(t, cursorRows, 1)
	encoded, err := json.Marshal(cursorRows[0])
	require.NoError(t, err)
	var cursorRow map[string]interface{}
	require.NoError(t, json.Unmarshal(encoded, &cursorRow))
	assertMessage(cursorRow)
	byID, err := repo.GetShopMessageByID(context.Background(), &bootstrap.User{UserID: "user-1"}, id)
	require.NoError(t, err)
	require.Equal(t, id, byID.ID)
	require.Equal(t, "persisted generation message", byID.Message)
	page := decodeSyncPage(t, doContractRequest(t, router, http.MethodGet, messageSyncPath(shopID, "initial", nil), nil, "user-1"))
	require.Len(t, page.Rows, 1)
	assertMessage(page.Rows[0])
}

// This explicit set is the tracked generated contract, not every user_pmcs table
// in the database. In particular, inspection_comments is currently untracked.
func trackedUserPMCSReferencePaths() []string {
	var paths []string
	for _, directory := range []string{"model", "table"} {
		for _, name := range []string{
			"checklists", "community_releases", "community_sources", "community_votes",
			"content_uuid_reservations", "faults", "inspections", "items", "notices",
			"procedure_steps", "revision_models", "revisions", "section_models", "sections",
			"subscriptions", "sync_state",
		} {
			paths = append(paths, ".gen/miltech_ng/public/"+directory+"/user_pmcs_"+name+".go")
		}
	}
	return paths
}

func validateUserPMCSReferenceHashes(references map[string]string) error {
	paths := trackedUserPMCSReferencePaths()
	if len(references) != len(paths) {
		return fmt.Errorf("expected %d tracked user_pmcs references, got %d", len(paths), len(references))
	}
	for _, name := range paths {
		digest, err := hex.DecodeString(references[name])
		if err != nil || len(digest) != sha256.Size {
			return fmt.Errorf("missing or malformed tracked user_pmcs reference: %s", name)
		}
	}
	return nil
}

func TestShopMessageGenerationReferencesRejectIncompleteOrMalformedContracts(t *testing.T) {
	for _, name := range []string{"complete", "empty", "partial", "malformed hash", "redirected path", "unexpected contract"} {
		t.Run(name, func(t *testing.T) {
			references := map[string]string{}
			for _, path := range trackedUserPMCSReferencePaths() {
				references[path] = strings.Repeat("a", 64)
			}
			const checklist = ".gen/miltech_ng/public/model/user_pmcs_checklists.go"
			switch name {
			case "empty":
				references = nil
			case "partial":
				delete(references, checklist)
			case "malformed hash":
				references[checklist] = strings.Repeat("z", 64)
			case "redirected path":
				delete(references, checklist)
				references["../user_pmcs_checklists.go"] = strings.Repeat("a", 64)
			case "unexpected contract":
				delete(references, checklist)
				references[".gen/miltech_ng/public/model/user_pmcs_inspection_comments.go"] = strings.Repeat("a", 64)
			}
			if name == "complete" {
				require.NoError(t, validateUserPMCSReferenceHashes(references))
			} else {
				require.Error(t, validateUserPMCSReferenceHashes(references))
			}
		})
	}
}

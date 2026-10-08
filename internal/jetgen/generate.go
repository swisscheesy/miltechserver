// Package jetgen owns tagged, canonical generation for startup and the CLI.
package jetgen

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/go-jet/jet/v2/generator/postgres"
	"golang.org/x/sys/unix"
	"os"
	"path/filepath"
	"regexp"
	"runtime/debug"
	"time"
)

type Config struct{ Schema, OutputDirectory string }

// Manifest records the real source, not the canonical output namespace.
// Generation does not rebuild or certify the running binary.
type Manifest struct {
	Database         string    `json:"database"`
	User             string    `json:"user"`
	Address          string    `json:"address"`
	Port             int       `json:"port"`
	ServerVersion    string    `json:"server_version"`
	Schema           string    `json:"schema"`
	CatalogSHA256    string    `json:"catalog_sha256"`
	GeneratedAt      time.Time `json:"generated_at"`
	GeneratorVersion string    `json:"generator_version"`
	BuildRevision    string    `json:"build_revision"`
	Compatibility    string    `json:"compatibility"`
}

var schemaName = regexp.MustCompile(`^[a-z_][a-z0-9_]*$`)

func Generate(db *sql.DB, cfg Config) error {
	if db == nil || !schemaName.MatchString(cfg.Schema) || cfg.OutputDirectory == "" {
		return errors.New("invalid Jet generation configuration")
	}
	output, err := filepath.Abs(cfg.OutputDirectory)
	if err != nil || output == string(filepath.Separator) {
		return errors.New("invalid Jet output directory")
	}
	if err := rejectSymlinks(output); err != nil {
		return err
	}
	if err := os.MkdirAll(output, 0755); err != nil {
		return errors.New("Jet output directory is not writable")
	}
	output, err = filepath.EvalSymlinks(output)
	if err != nil {
		return errors.New("unable to resolve Jet output directory")
	}
	fd, err := unix.Open(filepath.Join(output, ".jetgen.lock"), unix.O_CREAT|unix.O_RDWR|unix.O_NOFOLLOW, 0600)
	if err != nil {
		return errors.New("unable to open Jet output lock")
	}
	defer unix.Close(fd)
	if err := unix.Flock(fd, unix.LOCK_EX); err != nil {
		return errors.New("unable to lock Jet output")
	}
	defer unix.Flock(fd, unix.LOCK_UN)
	if err := recoverPublication(filepath.Join(output, "miltech_ng", cfg.Schema), os.Rename); err != nil {
		return err
	}
	stage, err := os.MkdirTemp(output, ".jetgen-stage-")
	if err != nil {
		return errors.New("Jet output directory is not writable")
	}
	defer os.RemoveAll(stage)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	manifest, err := inspectSource(ctx, db, cfg.Schema)
	if err != nil {
		return err
	}
	if err := postgres.GenerateDB(db, cfg.Schema, filepath.Join(stage, "miltech_ng"), JSONTaggedTemplate()); err != nil {
		return errors.New("Jet schema generation failed")
	}
	after, err := inspectSource(ctx, db, cfg.Schema)
	if err != nil {
		return err
	}
	if manifest.CatalogSHA256 != after.CatalogSHA256 || manifest.Database != after.Database || manifest.User != after.User || manifest.Address != after.Address || manifest.Port != after.Port {
		return errors.New("Jet source changed during generation")
	}
	manifest.GeneratedAt = time.Now().UTC()
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return errors.New("unable to encode Jet manifest")
	}
	stagedSchema := filepath.Join(stage, "miltech_ng", cfg.Schema)
	if err := os.WriteFile(filepath.Join(stagedSchema, "generation-manifest.json"), append(data, '\n'), 0644); err != nil {
		return errors.New("unable to write Jet manifest")
	}
	return publishSchema(stagedSchema, filepath.Join(output, "miltech_ng", cfg.Schema), os.Rename)
}

// Ancestors may include OS aliases such as macOS /var; the selected output and
// publication entries themselves must be real directories, never redirected links.
func rejectSymlinks(path string) error {
	info, err := os.Lstat(path)
	if err != nil && !os.IsNotExist(err) {
		return errors.New("unable to inspect Jet output")
	}
	if err == nil && info.Mode()&os.ModeSymlink != 0 {
		return errors.New("Jet output must not contain symbolic links")
	}
	return nil
}

// The output lock serializes publishers; the two renames are not an atomic exchange.
// Failed publication restores the previous complete schema. Interrupted publication
// retains its backup so the next publisher can recover it.
func publishSchema(staged, target string, rename func(string, string) error) error {
	if err := recoverPublication(target, rename); err != nil {
		return err
	}
	backup := target + ".jetgen-backup"

	hasPrevious := false
	if _, err := os.Stat(target); err == nil {
		if err := rename(target, backup); err != nil {
			return errors.New("unable to preserve prior Jet output")
		}
		hasPrevious = true
	} else if !os.IsNotExist(err) {
		return errors.New("unable to inspect Jet output")
	}
	if err := rename(staged, target); err != nil {
		if hasPrevious {
			if err := rename(backup, target); err != nil {
				return errors.New("Jet publication failed; prior output retained in backup; recovery required")
			}
		}
		return errors.New("Jet publication failed; prior output restored")
	}
	if hasPrevious {
		if err := os.RemoveAll(backup); err != nil {
			return errors.New("Jet output published but backup cleanup failed")
		}
	}
	return nil
}

func recoverPublication(target string, rename func(string, string) error) error {
	if err := rejectSymlinks(filepath.Dir(target)); err != nil {
		return err
	}
	if err := rejectSymlinks(target); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		return errors.New("unable to prepare Jet publication")
	}
	backup := target + ".jetgen-backup"
	if err := rejectSymlinks(backup); err != nil {
		return err
	}
	if _, err := os.Stat(backup); err == nil {
		if _, err := os.Stat(target); os.IsNotExist(err) {
			if err := rename(backup, target); err != nil {
				return errors.New("unable to recover prior Jet output")
			}
		} else if err != nil {
			return errors.New("unable to inspect prior Jet output")
		} else if err := os.RemoveAll(backup); err != nil {
			return errors.New("unable to clean prior Jet output")
		}
	} else if !os.IsNotExist(err) {
		return errors.New("unable to inspect Jet backup")
	}
	return nil
}

func inspectSource(ctx context.Context, db *sql.DB, schema string) (Manifest, error) {
	m := Manifest{Schema: schema, GeneratorVersion: "go-jet/v2 v2.13.0", Compatibility: "legacy shop_messages pre/post-018; additional release readiness gates required"}
	if err := db.QueryRowContext(ctx, `SELECT current_database(),current_user,COALESCE(inet_server_addr()::text,'local-socket'),COALESCE(inet_server_port(),current_setting('port')::integer),current_setting('server_version')`).Scan(&m.Database, &m.User, &m.Address, &m.Port, &m.ServerVersion); err != nil {
		return m, errors.New("unable to verify Jet source identity")
	}
	var usable bool
	if err := db.QueryRowContext(ctx, `SELECT has_schema_privilege(current_user,$1,'USAGE')`, schema).Scan(&usable); err != nil || !usable {
		return m, errors.New("Jet source schema is unavailable")
	}
	rows, err := db.QueryContext(ctx, `SELECT column_name,udt_name,is_nullable FROM information_schema.columns WHERE table_schema=$1 AND table_name='shop_messages'`, schema)
	if err != nil {
		return m, errors.New("unable to read Jet compatibility catalog")
	}
	columns := map[string]string{}
	for rows.Next() {
		var name, kind, nullable string
		if err := rows.Scan(&name, &kind, &nullable); err != nil {
			rows.Close()
			return m, errors.New("unable to read Jet compatibility catalog")
		}
		columns[name] = kind + ":" + nullable
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return m, errors.New("unable to read Jet compatibility catalog")
	}
	required := map[string]string{"id": "text:NO", "shop_id": "text:NO", "user_id": "text:NO", "message": "text:NO", "created_at": "timestamptz:YES", "updated_at": "timestamptz:YES", "is_edited": "bool:YES", "parent_id": "text:YES"}
	for name, kind := range required {
		if columns[name] != kind {
			return m, errors.New("Jet source is incompatible with legacy message columns")
		}
	}
	if kind, exists := columns["insertion_number"]; exists && kind != "int8:YES" && kind != "int8:NO" {
		return m, errors.New("Jet source has incompatible message insertion number")
	}
	var catalog string
	if err := db.QueryRowContext(ctx, catalogQuery, schema).Scan(&catalog); err != nil {
		return m, errors.New("unable to fingerprint Jet schema catalog")
	}
	hash := sha256.Sum256([]byte(catalog))
	m.CatalogSHA256 = hex.EncodeToString(hash[:])
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range info.Settings {
			if setting.Key == "vcs.revision" {
				m.BuildRevision = setting.Value
			}
		}
	}
	return m, nil
}

// The digest covers relation/column definitions, defaults, constraints, indexes,
// functions, triggers, enums and sequences; it is provenance, not an exact-match gate.
// relkind and tgenabled are the internal "char" type: text || "char" is ambiguous
// on Postgres 15+ ("operator is not unique"), so both need an explicit ::text.
const catalogQuery = `SELECT COALESCE(string_agg(record,E'\n' ORDER BY record),'') FROM (
 SELECT 'column:'||c.relname||':'||a.attnum||':'||a.attname||':'||format_type(a.atttypid,a.atttypmod)||':'||a.attnotnull||':'||COALESCE(pg_get_expr(d.adbin,d.adrelid),'') record FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace JOIN pg_attribute a ON a.attrelid=c.oid LEFT JOIN pg_attrdef d ON d.adrelid=c.oid AND d.adnum=a.attnum WHERE n.nspname=$1 AND a.attnum>0 AND NOT a.attisdropped
 UNION ALL SELECT 'relation:'||c.relname||':'||c.relkind::text||':'||CASE WHEN c.relkind IN ('v','m') THEN pg_get_viewdef(c.oid) ELSE '' END FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname=$1
 UNION ALL SELECT 'constraint:'||c.conname||':'||pg_get_constraintdef(c.oid) FROM pg_constraint c JOIN pg_namespace n ON n.oid=c.connamespace WHERE n.nspname=$1
 UNION ALL SELECT 'index:'||indexname||':'||indexdef FROM pg_indexes WHERE schemaname=$1
 UNION ALL SELECT 'function:'||p.proname||':'||pg_get_functiondef(p.oid) FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace WHERE n.nspname=$1 AND p.prokind IN ('f','p')
 UNION ALL SELECT 'trigger:'||c.relname||':'||t.tgenabled::text||':'||pg_get_triggerdef(t.oid) FROM pg_trigger t JOIN pg_class c ON c.oid=t.tgrelid JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname=$1 AND NOT t.tgisinternal
 UNION ALL SELECT 'enum:'||t.typname||':'||e.enumsortorder||':'||e.enumlabel FROM pg_type t JOIN pg_namespace n ON n.oid=t.typnamespace JOIN pg_enum e ON e.enumtypid=t.oid WHERE n.nspname=$1
 UNION ALL SELECT 'sequence:'||sequencename||':'||data_type||':'||start_value||':'||min_value||':'||max_value||':'||increment_by||':'||cycle FROM pg_sequences WHERE schemaname=$1
 ) catalog`

package pmcs_sbs_progress_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type pmcsColumnSignature struct {
	name         string
	dataType     string
	notNull      bool
	defaultValue string
}

func TestUserPmcsPersistenceSchema(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	requirePmcsSbsTestDatabase(t, ctx, testDB)

	requireTargetRelations(t, ctx, testDB)
	requireTargetColumns(t, ctx, testDB)
	requireTargetConstraints(t, ctx, testDB)
	requireTargetIndexes(t, ctx, testDB)
}

func requireTargetRelations(t *testing.T, ctx context.Context, db *sql.DB) {
	t.Helper()

	rows, err := db.QueryContext(ctx, `
		SELECT class.relname, class.relkind
		FROM pg_catalog.pg_class AS class
		JOIN pg_catalog.pg_namespace AS namespace
		  ON namespace.oid = class.relnamespace
		WHERE namespace.nspname = 'public'
		  AND class.relname IN (
		    'user_pmcs_inspections',
		    'user_pmcs_faults',
		    'user_pmcs_inspection_comments',
		    'pmcs_sbs_inspections',
		    'pmcs_sbs_faults',
		    'pmcs_sbs_inspection_comments'
		  )
		ORDER BY class.relname`)
	require.NoError(t, err)
	defer rows.Close()

	relations := make(map[string]string)
	for rows.Next() {
		var name string
		var kind string
		require.NoError(t, rows.Scan(&name, &kind))
		relations[name] = kind
	}
	require.NoError(t, rows.Err())
	require.Equal(t, map[string]string{
		"user_pmcs_faults":              "r",
		"user_pmcs_inspection_comments": "r",
		"user_pmcs_inspections":         "r",
	}, relations)
}

func requireTargetColumns(t *testing.T, ctx context.Context, db *sql.DB) {
	t.Helper()

	expected := map[string][]pmcsColumnSignature{
		"user_pmcs_inspections": {
			{name: "id", dataType: "uuid", notNull: true},
			{name: "equipment_id", dataType: "text", notNull: true},
			{name: "guide_manual", dataType: "text"},
			{name: "performed_date", dataType: "timestamp with time zone", notNull: true},
			{name: "performed_by", dataType: "text"},
			{name: "created_at", dataType: "timestamp with time zone", notNull: true, defaultValue: "now()"},
			{name: "updated_at", dataType: "timestamp with time zone", notNull: true, defaultValue: "now()"},
			{name: "notes", dataType: "text"},
			{name: "source_type", dataType: "text", notNull: true},
			{name: "custom_checklist_id", dataType: "uuid"},
			{name: "custom_revision_id", dataType: "uuid"},
			{name: "custom_revision_number", dataType: "integer"},
			{name: "custom_checklist_name", dataType: "text"},
		},
		"user_pmcs_faults": {
			{name: "pmcs_id", dataType: "uuid", notNull: true},
			{name: "section_id", dataType: "text", notNull: true},
			{name: "item_index", dataType: "integer", notNull: true},
			{name: "item_no", dataType: "text", notNull: true},
			{name: "status", dataType: "text", notNull: true},
			{name: "fault_text", dataType: "text", notNull: true},
			{name: "corrective_action", dataType: "text", notNull: true, defaultValue: "''::text"},
			{name: "created_at", dataType: "timestamp with time zone", notNull: true, defaultValue: "now()"},
			{name: "updated_at", dataType: "timestamp with time zone", notNull: true, defaultValue: "now()"},
			{name: "section_title", dataType: "text"},
		},
		"user_pmcs_inspection_comments": {
			{name: "id", dataType: "uuid", notNull: true, defaultValue: "gen_random_uuid()"},
			{name: "pmcs_id", dataType: "uuid", notNull: true},
			{name: "author_id", dataType: "text", notNull: true},
			{name: "text", dataType: "text", notNull: true},
			{name: "created_at", dataType: "timestamp with time zone", notNull: true, defaultValue: "now()"},
			{name: "updated_at", dataType: "timestamp with time zone"},
		},
	}

	for tableName, expectedColumns := range expected {
		rows, err := db.QueryContext(ctx, `
			SELECT attribute.attname,
			       pg_catalog.format_type(attribute.atttypid, attribute.atttypmod),
			       attribute.attnotnull,
			       COALESCE(pg_catalog.pg_get_expr(definition.adbin, definition.adrelid), '')
			FROM pg_catalog.pg_attribute AS attribute
			LEFT JOIN pg_catalog.pg_attrdef AS definition
			  ON definition.adrelid = attribute.attrelid
			 AND definition.adnum = attribute.attnum
			WHERE attribute.attrelid = $1::regclass
			  AND attribute.attnum > 0
			  AND NOT attribute.attisdropped
			ORDER BY attribute.attnum`, "public."+tableName)
		require.NoError(t, err)

		var actualColumns []pmcsColumnSignature
		for rows.Next() {
			var column pmcsColumnSignature
			require.NoError(t, rows.Scan(
				&column.name,
				&column.dataType,
				&column.notNull,
				&column.defaultValue,
			))
			actualColumns = append(actualColumns, column)
		}
		require.NoError(t, rows.Err())
		require.NoError(t, rows.Close())
		require.Equal(t, expectedColumns, actualColumns, tableName)
	}
}

func requireTargetConstraints(t *testing.T, ctx context.Context, db *sql.DB) {
	t.Helper()

	expected := map[string]string{
		"user_pmcs_inspections_pkey":                        "PRIMARY KEY (id)",
		"fk_user_pmcs_inspections_equipment_id":             "FOREIGN KEY (equipment_id) REFERENCES shop_vehicle(id) ON UPDATE CASCADE ON DELETE CASCADE",
		"fk_user_pmcs_inspections_performed_by":             "FOREIGN KEY (performed_by) REFERENCES users(uid) ON UPDATE CASCADE ON DELETE SET NULL",
		"user_pmcs_inspections_equipment_id_nonblank_check": "CHECK (btrim(equipment_id) <> ''::text)",
		"user_pmcs_inspections_source_shape_check":          `CHECK (source_type = 'guide'::text AND guide_manual IS NOT NULL AND guide_manual = btrim(guide_manual) AND guide_manual ~~ 'pmcs_sbs/%'::text AND "right"(guide_manual, 5) = '.json'::text AND custom_checklist_id IS NULL AND custom_revision_id IS NULL AND custom_revision_number IS NULL AND custom_checklist_name IS NULL OR source_type = 'custom'::text AND guide_manual IS NULL AND custom_checklist_id IS NOT NULL AND custom_revision_id IS NOT NULL AND custom_revision_number IS NOT NULL AND custom_revision_number >= 0 AND custom_checklist_name IS NOT NULL AND custom_checklist_name = btrim(custom_checklist_name) AND btrim(custom_checklist_name) <> ''::text)`,
		"user_pmcs_inspections_source_type_check":           "CHECK (source_type = ANY (ARRAY['guide'::text, 'custom'::text]))",
		"user_pmcs_faults_pkey":                             "PRIMARY KEY (pmcs_id, section_id, item_index)",
		"fk_user_pmcs_faults_pmcs_id":                       "FOREIGN KEY (pmcs_id) REFERENCES user_pmcs_inspections(id) ON UPDATE CASCADE ON DELETE CASCADE",
		"user_pmcs_faults_item_index_check":                 "CHECK (item_index >= 0)",
		"user_pmcs_faults_nonblank_fields_check":            "CHECK (btrim(section_id) <> ''::text AND btrim(item_no) <> ''::text AND btrim(fault_text) <> ''::text)",
		"user_pmcs_faults_status_check":                     "CHECK (status = ANY (ARRAY['x'::text, 'slash'::text, 'dash'::text]))",
		"user_pmcs_inspection_comments_pkey":                "PRIMARY KEY (id)",
		"fk_user_pmcs_inspection_comments_author_id":        "FOREIGN KEY (author_id) REFERENCES users(uid) ON UPDATE CASCADE",
		"fk_user_pmcs_inspection_comments_pmcs_id":          "FOREIGN KEY (pmcs_id) REFERENCES user_pmcs_inspections(id) ON UPDATE CASCADE ON DELETE CASCADE",
		"user_pmcs_inspection_comments_nonblank_check":      "CHECK (btrim(text) <> ''::text)",
	}

	rows, err := db.QueryContext(ctx, `
		SELECT constraint_record.conname,
		       pg_catalog.pg_get_constraintdef(constraint_record.oid, true)
		FROM pg_catalog.pg_constraint AS constraint_record
		JOIN pg_catalog.pg_class AS class
		  ON class.oid = constraint_record.conrelid
		JOIN pg_catalog.pg_namespace AS namespace
		  ON namespace.oid = class.relnamespace
		WHERE namespace.nspname = 'public'
		  AND class.relname IN (
		    'user_pmcs_inspections',
		    'user_pmcs_faults',
		    'user_pmcs_inspection_comments'
		  )`)
	require.NoError(t, err)
	defer rows.Close()

	actual := make(map[string]string)
	for rows.Next() {
		var name string
		var definition string
		require.NoError(t, rows.Scan(&name, &definition))
		actual[name] = definition
	}
	require.NoError(t, rows.Err())
	require.Equal(t, expected, actual)
}

func requireTargetIndexes(t *testing.T, ctx context.Context, db *sql.DB) {
	t.Helper()

	expected := map[string]string{
		"user_pmcs_inspections_pkey":                    "CREATE UNIQUE INDEX user_pmcs_inspections_pkey ON public.user_pmcs_inspections USING btree (id)",
		"user_pmcs_inspections_equipment_performed_idx": "CREATE INDEX user_pmcs_inspections_equipment_performed_idx ON public.user_pmcs_inspections USING btree (equipment_id, performed_date DESC)",
		"user_pmcs_inspections_performed_by_idx":        "CREATE INDEX user_pmcs_inspections_performed_by_idx ON public.user_pmcs_inspections USING btree (performed_by)",
		"user_pmcs_faults_pkey":                         "CREATE UNIQUE INDEX user_pmcs_faults_pkey ON public.user_pmcs_faults USING btree (pmcs_id, section_id, item_index)",
		"user_pmcs_inspection_comments_pkey":            "CREATE UNIQUE INDEX user_pmcs_inspection_comments_pkey ON public.user_pmcs_inspection_comments USING btree (id)",
		"user_pmcs_inspection_comments_pmcs_id_idx":     "CREATE INDEX user_pmcs_inspection_comments_pmcs_id_idx ON public.user_pmcs_inspection_comments USING btree (pmcs_id, created_at)",
		"user_pmcs_inspection_comments_author_id_idx":   "CREATE INDEX user_pmcs_inspection_comments_author_id_idx ON public.user_pmcs_inspection_comments USING btree (author_id)",
	}

	rows, err := db.QueryContext(ctx, `
		SELECT indexname, indexdef
		FROM pg_catalog.pg_indexes
		WHERE schemaname = 'public'
		  AND tablename IN (
		    'user_pmcs_inspections',
		    'user_pmcs_faults',
		    'user_pmcs_inspection_comments'
		  )`)
	require.NoError(t, err)
	defer rows.Close()

	actual := make(map[string]string)
	for rows.Next() {
		var name string
		var definition string
		require.NoError(t, rows.Scan(&name, &definition))
		actual[name] = definition
	}
	require.NoError(t, rows.Err())
	require.Equal(t, expected, actual)
}

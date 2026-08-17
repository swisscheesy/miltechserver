package pmcs_sbs_progress

import (
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type renamePair struct {
	forward  string
	rollback string
}

func TestTableRenameMigrationContract(t *testing.T) {
	t.Parallel()

	forward := readMigration(t, "../../migrations/014_rename_pmcs_tables.sql")
	rollback := readMigration(t, "../../migrations/014_rollback_rename_pmcs_tables.sql")

	requireMigrationEnvelope(t, forward, `LOCK TABLE
		public.pmcs_sbs_inspections,
		public.pmcs_sbs_faults,
		public.pmcs_sbs_inspection_comments
		IN ACCESS EXCLUSIVE MODE;`)
	requireMigrationEnvelope(t, rollback, `LOCK TABLE
		public.user_pmcs_inspections,
		public.user_pmcs_faults,
		public.user_pmcs_inspection_comments
		IN ACCESS EXCLUSIVE MODE;`)

	renames := []renamePair{
		{
			forward:  "ALTER TABLE public.pmcs_sbs_inspections RENAME TO user_pmcs_inspections;",
			rollback: "ALTER TABLE public.user_pmcs_inspections RENAME TO pmcs_sbs_inspections;",
		},
		{
			forward:  "ALTER TABLE public.pmcs_sbs_faults RENAME TO user_pmcs_faults;",
			rollback: "ALTER TABLE public.user_pmcs_faults RENAME TO pmcs_sbs_faults;",
		},
		{
			forward:  "ALTER TABLE public.pmcs_sbs_inspection_comments RENAME TO user_pmcs_inspection_comments;",
			rollback: "ALTER TABLE public.user_pmcs_inspection_comments RENAME TO pmcs_sbs_inspection_comments;",
		},
		constraintRename(
			"user_pmcs_inspections",
			"pmcs_sbs_inspections_pkey",
			"user_pmcs_inspections_pkey",
		),
		constraintRename(
			"user_pmcs_inspections",
			"fk_pmcs_sbs_inspections_equipment_id",
			"fk_user_pmcs_inspections_equipment_id",
		),
		constraintRename(
			"user_pmcs_inspections",
			"fk_pmcs_sbs_inspections_performed_by",
			"fk_user_pmcs_inspections_performed_by",
		),
		constraintRename(
			"user_pmcs_inspections",
			"pmcs_sbs_inspections_equipment_id_nonblank_check",
			"user_pmcs_inspections_equipment_id_nonblank_check",
		),
		constraintRename(
			"user_pmcs_inspections",
			"pmcs_sbs_inspections_source_shape_check",
			"user_pmcs_inspections_source_shape_check",
		),
		constraintRename(
			"user_pmcs_inspections",
			"pmcs_sbs_inspections_source_type_check",
			"user_pmcs_inspections_source_type_check",
		),
		constraintRename(
			"user_pmcs_faults",
			"pmcs_sbs_faults_pkey",
			"user_pmcs_faults_pkey",
		),
		constraintRename(
			"user_pmcs_faults",
			"fk_pmcs_sbs_faults_pmcs_id",
			"fk_user_pmcs_faults_pmcs_id",
		),
		constraintRename(
			"user_pmcs_faults",
			"pmcs_sbs_faults_item_index_check",
			"user_pmcs_faults_item_index_check",
		),
		constraintRename(
			"user_pmcs_faults",
			"pmcs_sbs_faults_nonblank_fields_check",
			"user_pmcs_faults_nonblank_fields_check",
		),
		constraintRename(
			"user_pmcs_faults",
			"pmcs_sbs_faults_status_check",
			"user_pmcs_faults_status_check",
		),
		constraintRename(
			"user_pmcs_inspection_comments",
			"pmcs_sbs_inspection_comments_pkey",
			"user_pmcs_inspection_comments_pkey",
		),
		constraintRename(
			"user_pmcs_inspection_comments",
			"fk_pmcs_sbs_inspection_comments_author_id",
			"fk_user_pmcs_inspection_comments_author_id",
		),
		constraintRename(
			"user_pmcs_inspection_comments",
			"fk_pmcs_sbs_inspection_comments_pmcs_id",
			"fk_user_pmcs_inspection_comments_pmcs_id",
		),
		constraintRename(
			"user_pmcs_inspection_comments",
			"pmcs_sbs_inspection_comments_nonblank_check",
			"user_pmcs_inspection_comments_nonblank_check",
		),
		{
			forward:  "ALTER INDEX public.idx_pmcs_sbs_inspections_equipment_performed RENAME TO user_pmcs_inspections_equipment_performed_idx;",
			rollback: "ALTER INDEX public.user_pmcs_inspections_equipment_performed_idx RENAME TO idx_pmcs_sbs_inspections_equipment_performed;",
		},
		{
			forward:  "ALTER INDEX public.idx_pmcs_sbs_inspection_comments_pmcs_id RENAME TO user_pmcs_inspection_comments_pmcs_id_idx;",
			rollback: "ALTER INDEX public.user_pmcs_inspection_comments_pmcs_id_idx RENAME TO idx_pmcs_sbs_inspection_comments_pmcs_id;",
		},
	}

	for _, rename := range renames {
		require.Equal(t, 1, strings.Count(forward, rename.forward), rename.forward)
		require.Equal(t, 1, strings.Count(rollback, rename.rollback), rename.rollback)
	}

	createdIndexes := []renamePair{
		{
			forward:  "CREATE INDEX user_pmcs_inspections_performed_by_idx ON public.user_pmcs_inspections (performed_by);",
			rollback: "DROP INDEX public.user_pmcs_inspections_performed_by_idx;",
		},
		{
			forward:  "CREATE INDEX user_pmcs_inspection_comments_author_id_idx ON public.user_pmcs_inspection_comments (author_id);",
			rollback: "DROP INDEX public.user_pmcs_inspection_comments_author_id_idx;",
		},
	}
	for _, index := range createdIndexes {
		require.Equal(t, 1, strings.Count(normalizeWhitespace(forward), normalizeWhitespace(index.forward)), index.forward)
		require.Equal(t, 1, strings.Count(normalizeWhitespace(rollback), normalizeWhitespace(index.rollback)), index.rollback)
	}

	for _, migration := range []string{forward, rollback} {
		require.Contains(t, migration, "class.relkind = 'r'")
		require.Contains(t, migration, "pg_catalog.to_regclass")
		require.NotRegexp(t, regexp.MustCompile(`(?i)\bDROP\s+TABLE\b`), migration)
		require.NotRegexp(t, regexp.MustCompile(`(?i)\bCREATE\s+TABLE\b`), migration)
		require.NotRegexp(t, regexp.MustCompile(`(?i)\bCASCADE\b`), migration)
		require.NotRegexp(t, regexp.MustCompile(`(?i)\bIF\s+EXISTS\b`), migration)
		require.NotRegexp(t, regexp.MustCompile(`(?i)ALTER\s+INDEX[^;]*_pkey`), migration)
	}
}

func constraintRename(tableName string, legacyName string, targetName string) renamePair {
	return renamePair{
		forward:  "ALTER TABLE public." + tableName + " RENAME CONSTRAINT " + legacyName + " TO " + targetName + ";",
		rollback: "ALTER TABLE public." + tableName + " RENAME CONSTRAINT " + targetName + " TO " + legacyName + ";",
	}
}

func readMigration(t *testing.T, path string) string {
	t.Helper()

	contents, err := os.ReadFile(path)
	require.NoError(t, err)
	return string(contents)
}

func requireMigrationEnvelope(t *testing.T, migration string, lockStatement string) {
	t.Helper()

	require.Equal(t, 1, strings.Count(migration, "BEGIN;"))
	require.Equal(t, 1, strings.Count(migration, "COMMIT;"))
	require.Contains(t, migration, "SET LOCAL lock_timeout")
	require.Contains(t, migration, "SET LOCAL statement_timeout")
	require.Equal(t, 1, strings.Count(normalizeWhitespace(migration), normalizeWhitespace(lockStatement)))
}

func normalizeWhitespace(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

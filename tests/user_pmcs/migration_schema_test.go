package user_pmcs_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUserPmcsSchemaIntegrity(t *testing.T) {
	requiredTables := []string{
		"user_pmcs_sync_state",
		"user_pmcs_checklists",
		"user_pmcs_revisions",
		"user_pmcs_revision_models",
		"user_pmcs_sections",
		"user_pmcs_section_models",
		"user_pmcs_items",
		"user_pmcs_notices",
		"user_pmcs_procedure_steps",
		"user_pmcs_community_sources",
		"user_pmcs_community_releases",
		"user_pmcs_subscriptions",
	}
	for _, tableName := range requiredTables {
		t.Run("table/"+tableName, func(t *testing.T) {
			var exists bool
			err := testDB.QueryRow(`
				SELECT EXISTS (
					SELECT 1
					FROM information_schema.tables
					WHERE table_schema = 'public' AND table_name = $1
				)`, tableName).Scan(&exists)
			require.NoError(t, err)
			require.True(t, exists, "required table %s is missing", tableName)
		})
	}

	restrictiveForeignKeys := []struct {
		constraintName    string
		childTable        string
		childColumns      string
		referencedTable   string
		referencedColumns string
	}{
		{
			constraintName:    "fk_user_pmcs_community_releases_revision",
			childTable:        "user_pmcs_community_releases",
			childColumns:      "checklist_id,revision_id",
			referencedTable:   "user_pmcs_revisions",
			referencedColumns: "checklist_id,id",
		},
		{
			constraintName:    "fk_user_pmcs_community_sources_current_release",
			childTable:        "user_pmcs_community_sources",
			childColumns:      "checklist_id,current_release_revision_id",
			referencedTable:   "user_pmcs_community_releases",
			referencedColumns: "checklist_id,revision_id",
		},
		{
			constraintName:    "fk_user_pmcs_subscriptions_installed_release",
			childTable:        "user_pmcs_subscriptions",
			childColumns:      "checklist_id,installed_revision_id",
			referencedTable:   "user_pmcs_community_releases",
			referencedColumns: "checklist_id,revision_id",
		},
	}
	for _, expected := range restrictiveForeignKeys {
		t.Run("restrictive_foreign_key/"+expected.constraintName, func(t *testing.T) {
			var (
				deleteAction      string
				childColumns      string
				referencedTable   string
				referencedColumns string
			)
			err := testDB.QueryRow(`
				SELECT
					CASE constraint_definition.confdeltype
						WHEN 'r' THEN 'RESTRICT'
						ELSE constraint_definition.confdeltype::text
					END,
					string_agg(child_column.attname, ',' ORDER BY child_key.ordinality),
					referenced_table.relname,
					string_agg(referenced_column.attname, ',' ORDER BY child_key.ordinality)
				FROM pg_constraint AS constraint_definition
				JOIN pg_class AS child_table
					ON child_table.oid = constraint_definition.conrelid
				JOIN LATERAL unnest(constraint_definition.conkey)
					WITH ORDINALITY AS child_key(attnum, ordinality) ON true
				JOIN pg_attribute AS child_column
					ON child_column.attrelid = child_table.oid
					AND child_column.attnum = child_key.attnum
				JOIN pg_class AS referenced_table
					ON referenced_table.oid = constraint_definition.confrelid
				JOIN LATERAL unnest(constraint_definition.confkey)
					WITH ORDINALITY AS referenced_key(attnum, ordinality)
					ON referenced_key.ordinality = child_key.ordinality
				JOIN pg_attribute AS referenced_column
					ON referenced_column.attrelid = referenced_table.oid
					AND referenced_column.attnum = referenced_key.attnum
				WHERE constraint_definition.conname = $1
					AND child_table.relname = $2
				GROUP BY
					constraint_definition.confdeltype,
					referenced_table.relname`,
				expected.constraintName,
				expected.childTable,
			).Scan(
				&deleteAction,
				&childColumns,
				&referencedTable,
				&referencedColumns,
			)
			require.NoError(t, err)
			require.Equal(t, "RESTRICT", deleteAction)
			require.Equal(t, expected.childColumns, childColumns)
			require.Equal(t, expected.referencedTable, referencedTable)
			require.Equal(t, expected.referencedColumns, referencedColumns)
		})
	}

	partialIndexes := []struct {
		indexName         string
		state             string
		expectedPredicate string
	}{
		{
			indexName:         "user_pmcs_revisions_one_draft_idx",
			state:             "draft",
			expectedPredicate: "(state = 'draft'::text)",
		},
		{
			indexName:         "user_pmcs_revisions_one_published_idx",
			state:             "published",
			expectedPredicate: "(state = 'published'::text)",
		},
	}
	for _, expected := range partialIndexes {
		t.Run("partial_unique_index/"+expected.state, func(t *testing.T) {
			var (
				isUnique       bool
				indexedColumns string
				predicate      string
			)
			err := testDB.QueryRow(`
				SELECT
					index_definition.indisunique,
					string_agg(
						indexed_column.attname,
						',' ORDER BY index_key.ordinality
					),
					pg_get_expr(
						index_definition.indpred,
						index_definition.indrelid
					)
				FROM pg_index AS index_definition
				JOIN pg_class AS index_class
					ON index_class.oid = index_definition.indexrelid
				JOIN pg_class AS table_class
					ON table_class.oid = index_definition.indrelid
				JOIN pg_namespace AS table_schema
					ON table_schema.oid = table_class.relnamespace
				JOIN LATERAL unnest(index_definition.indkey)
					WITH ORDINALITY AS index_key(attnum, ordinality)
					ON index_key.ordinality <= index_definition.indnkeyatts
				JOIN pg_attribute AS indexed_column
					ON indexed_column.attrelid = table_class.oid
					AND indexed_column.attnum = index_key.attnum
				WHERE table_schema.nspname = 'public'
					AND table_class.relname = 'user_pmcs_revisions'
					AND index_class.relname = $1
				GROUP BY
					index_definition.indisunique,
					index_definition.indpred,
					index_definition.indrelid`,
				expected.indexName,
			).Scan(&isUnique, &indexedColumns, &predicate)
			require.NoError(t, err)
			require.True(t, isUnique)
			require.Equal(t, "checklist_id", indexedColumns)
			require.Equal(t, expected.expectedPredicate, predicate)
		})
	}

	rows, err := testDB.Query(`
		SELECT
			child_table.relname,
			constraint_definition.conname
		FROM pg_constraint AS constraint_definition
		JOIN pg_class AS child_table
			ON child_table.oid = constraint_definition.conrelid
		JOIN pg_namespace AS child_schema
			ON child_schema.oid = child_table.relnamespace
		WHERE constraint_definition.contype = 'f'
			AND child_schema.nspname = 'public'
			AND child_table.relname LIKE 'user_pmcs_%'
			AND NOT EXISTS (
				SELECT 1
				FROM pg_index AS child_index
				WHERE child_index.indrelid = constraint_definition.conrelid
					AND child_index.indisvalid
					AND child_index.indpred IS NULL
					AND child_index.indkey[0] = constraint_definition.conkey[1]
			)
		ORDER BY child_table.relname, constraint_definition.conname`)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, rows.Close())
	}()

	var missingLeadingIndexes []string
	for rows.Next() {
		var tableName, constraintName string
		require.NoError(t, rows.Scan(&tableName, &constraintName))
		missingLeadingIndexes = append(
			missingLeadingIndexes,
			tableName+"."+constraintName,
		)
	}
	require.NoError(t, rows.Err())
	require.Empty(
		t,
		missingLeadingIndexes,
		"every user PMCS child foreign key must have a non-partial leading index",
	)
}

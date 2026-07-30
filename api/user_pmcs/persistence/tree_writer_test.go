package persistence

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTreeBatchSizeNeverExceedsOneThousandBindParameters(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		columnsPerRow int
		expectedRows  int
		expectedBinds int
	}{
		{name: "revision models", columnsPerRow: 3, expectedRows: 333, expectedBinds: 999},
		{name: "sections", columnsPerRow: 4, expectedRows: 250, expectedBinds: 1000},
		{name: "items", columnsPerRow: 6, expectedRows: 166, expectedBinds: 996},
		{name: "notices", columnsPerRow: 5, expectedRows: 200, expectedBinds: 1000},
		{name: "steps", columnsPerRow: 5, expectedRows: 200, expectedBinds: 1000},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			rows := maxRowsPerBatch(testCase.columnsPerRow)
			require.Equal(t, testCase.expectedRows, rows)
			require.Equal(t, testCase.expectedBinds, rows*testCase.columnsPerRow)
			require.LessOrEqual(t, rows*testCase.columnsPerRow, maxBindParameters)
		})
	}
}

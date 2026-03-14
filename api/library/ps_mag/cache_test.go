package ps_mag

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestIssueCache_MissOnEmpty(t *testing.T) {
	c := newIssueCache(5 * time.Minute)
	_, ok := c.get()
	require.False(t, ok)
}

func TestIssueCache_HitAfterSet(t *testing.T) {
	c := newIssueCache(5 * time.Minute)
	issues := []PSMagIssueResponse{
		{Name: "test.pdf", IssueNumber: 1},
	}
	c.set(issues)

	got, ok := c.get()
	require.True(t, ok)
	require.Equal(t, issues, got)
}

func TestIssueCache_MissAfterExpiry(t *testing.T) {
	c := newIssueCache(1 * time.Millisecond)
	c.set([]PSMagIssueResponse{{Name: "test.pdf"}})

	time.Sleep(5 * time.Millisecond)

	_, ok := c.get()
	require.False(t, ok)
}

func TestIssueCache_SetOverwritesPrevious(t *testing.T) {
	c := newIssueCache(5 * time.Minute)
	c.set([]PSMagIssueResponse{{Name: "old.pdf"}})
	c.set([]PSMagIssueResponse{{Name: "new.pdf"}})

	got, ok := c.get()
	require.True(t, ok)
	require.Equal(t, "new.pdf", got[0].Name)
}

func TestIssueCache_GetReturnsCopy(t *testing.T) {
	// Mutating the returned slice must not corrupt the cache.
	c := newIssueCache(5 * time.Minute)
	c.set([]PSMagIssueResponse{{Name: "original.pdf"}})

	got, _ := c.get()
	got[0].Name = "mutated.pdf"

	got2, _ := c.get()
	require.Equal(t, "original.pdf", got2[0].Name)
}

func TestIssueCache_SetDoesNotAliasCallerSlice(t *testing.T) {
	// Mutating the slice passed to set() must not corrupt the cache.
	c := newIssueCache(5 * time.Minute)
	issues := []PSMagIssueResponse{{Name: "original.pdf"}}
	c.set(issues)

	// Mutate the original slice after set.
	issues[0].Name = "mutated.pdf"

	got, ok := c.get()
	require.True(t, ok)
	require.Equal(t, "original.pdf", got[0].Name)
}

package jetgen

import (
	"errors"
	"github.com/stretchr/testify/require"
	"miltechserver/internal/testsql"
	"os"
	"path/filepath"
	"testing"
)

func TestGenerationRejectsInvalidConfiguration(t *testing.T) {
	db := testsql.Open(t, &testsql.Behavior{})
	for _, cfg := range []Config{{}, {Schema: "../public", OutputDirectory: t.TempDir()}, {Schema: "public", OutputDirectory: "/"}} {
		require.Error(t, Generate(db, cfg))
	}
}
func TestGenerationCatalogFailurePreservesOutput(t *testing.T) {
	dir := t.TempDir()
	previous := filepath.Join(dir, "miltech_ng", "public")
	require.NoError(t, os.MkdirAll(previous, 0700))
	marker := filepath.Join(previous, "keep.go")
	require.NoError(t, os.WriteFile(marker, []byte("unchanged"), 0600))
	db := testsql.Open(t, &testsql.Behavior{QueryErr: errors.New("password=secret")})
	err := Generate(db, Config{Schema: "public", OutputDirectory: dir})
	require.Error(t, err)
	require.NotContains(t, err.Error(), "secret")
	content, err := os.ReadFile(marker)
	require.NoError(t, err)
	require.Equal(t, "unchanged", string(content))
}

func TestPublicationFailureRestoresPreviousSchema(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "public")
	stage := filepath.Join(root, "stage")
	require.NoError(t, os.Mkdir(target, 0700))
	require.NoError(t, os.Mkdir(stage, 0700))
	require.NoError(t, os.WriteFile(filepath.Join(target, "old.go"), []byte("old complete output"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(stage, "new.go"), []byte("new complete output"), 0600))
	err := publishSchema(stage, target, func(from, to string) error {
		if from == stage {
			return errors.New("private path failure")
		}
		return os.Rename(from, to)
	})
	require.Error(t, err)
	require.NotContains(t, err.Error(), "private path")
	data, err := os.ReadFile(filepath.Join(target, "old.go"))
	require.NoError(t, err)
	require.Equal(t, "old complete output", string(data))
	require.NoFileExists(t, filepath.Join(target, "new.go"))
	require.NoError(t, publishSchema(stage, target, os.Rename))
	require.FileExists(t, filepath.Join(target, "new.go"))
	require.NoFileExists(t, filepath.Join(target, "old.go"))
}
func TestPublicationRecoversInterruptedBackup(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "public")
	backup := target + ".jetgen-backup"
	stage := filepath.Join(root, "stage")
	require.NoError(t, os.Mkdir(backup, 0700))
	require.NoError(t, os.Mkdir(stage, 0700))
	require.NoError(t, os.WriteFile(filepath.Join(backup, "old.go"), []byte("old"), 0600))
	require.Error(t, publishSchema(stage, target, func(from, to string) error {
		if from == stage {
			return errors.New("fail")
		}
		return os.Rename(from, to)
	}))
	require.FileExists(t, filepath.Join(target, "old.go"))
}

func TestGenerationRestoresInterruptedOutputBeforeCatalogFailure(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "miltech_ng", "public")
	backup := target + ".jetgen-backup"
	require.NoError(t, os.MkdirAll(backup, 0700))
	require.NoError(t, os.WriteFile(filepath.Join(backup, "prior.go"), []byte("complete"), 0600))
	db := testsql.Open(t, &testsql.Behavior{QueryErr: errors.New("catalog unavailable")})
	require.Error(t, Generate(db, Config{Schema: "public", OutputDirectory: root}))
	data, err := os.ReadFile(filepath.Join(target, "prior.go"))
	require.NoError(t, err)
	require.Equal(t, "complete", string(data))
}

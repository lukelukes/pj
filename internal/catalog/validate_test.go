package catalog_test

import (
	"os"
	"path/filepath"
	"pj/internal/catalog"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateName(t *testing.T) {
	t.Run("empty name returns ErrEmptyName", func(t *testing.T) {
		assert.ErrorIs(t, catalog.ValidateName(""), catalog.ErrEmptyName)
	})

	t.Run("whitespace-only name returns ErrEmptyName", func(t *testing.T) {
		assert.ErrorIs(t, catalog.ValidateName("   \t\n  "), catalog.ErrEmptyName)
	})

	t.Run("valid name returns nil", func(t *testing.T) {
		assert.NoError(t, catalog.ValidateName("my-project"))
	})
}

func TestProject_ValidateAndNormalize(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("valid project passes validation", func(t *testing.T) {
		p := catalog.NewProject("myproject", tempDir)

		err := p.ValidateAndNormalize()

		assert.NoError(t, err)
	})

	t.Run("empty name fails validation", func(t *testing.T) {
		p := catalog.NewProject("", tempDir)

		err := p.ValidateAndNormalize()

		assert.ErrorIs(t, err, catalog.ErrEmptyName)
	})

	t.Run("whitespace-only name fails validation", func(t *testing.T) {
		p := catalog.NewProject("   \t\n  ", tempDir)

		err := p.ValidateAndNormalize()

		assert.ErrorIs(t, err, catalog.ErrEmptyName)
	})

	t.Run("relative path fails validation", func(t *testing.T) {
		p := catalog.NewProject("myproject", "relative/path")

		err := p.ValidateAndNormalize()

		assert.ErrorIs(t, err, catalog.ErrRelativePath)
		assert.Contains(t, err.Error(), "relative/path")
	})

	t.Run("non-existent absolute path fails validation", func(t *testing.T) {
		nonExistentPath := filepath.Join(tempDir, "nonexistent")
		p := catalog.NewProject("myproject", nonExistentPath)

		err := p.ValidateAndNormalize()

		assert.ErrorIs(t, err, catalog.ErrPathNotExist)
		assert.Contains(t, err.Error(), nonExistentPath)
	})

	t.Run("returns error for inaccessible path", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("permission test not reliable on Windows")
		}

		dir := t.TempDir()
		restrictedDir := filepath.Join(dir, "restricted")
		require.NoError(t, os.Mkdir(restrictedDir, 0o755))
		childDir := filepath.Join(restrictedDir, "child")
		require.NoError(t, os.Mkdir(childDir, 0o755))
		require.NoError(t, os.Chmod(restrictedDir, 0o000))
		t.Cleanup(func() { os.Chmod(restrictedDir, 0o755) })

		p := catalog.NewProject("test", childDir)
		err := p.ValidateAndNormalize()

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot access path")
	})
}

func TestYAMLCatalog_Add_Validation(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("rejects project with empty name", func(t *testing.T) {
		cat := newTestYAMLCatalog(t)
		p := catalog.NewProject("", tempDir)

		err := cat.Add(p)

		assert.ErrorIs(t, err, catalog.ErrEmptyName)
	})

	t.Run("rejects project with relative path", func(t *testing.T) {
		cat := newTestYAMLCatalog(t)
		p := catalog.NewProject("myproject", "relative/path")

		err := cat.Add(p)

		assert.ErrorIs(t, err, catalog.ErrRelativePath)
	})

	t.Run("rejects project with non-existent path", func(t *testing.T) {
		cat := newTestYAMLCatalog(t)
		nonExistentPath := filepath.Join(tempDir, "nonexistent")
		p := catalog.NewProject("myproject", nonExistentPath)

		err := cat.Add(p)

		assert.ErrorIs(t, err, catalog.ErrPathNotExist)
	})

	t.Run("accepts valid project", func(t *testing.T) {
		cat := newTestYAMLCatalog(t)
		p := catalog.NewProject("myproject", tempDir)

		err := cat.Add(p)

		assert.NoError(t, err)
	})
}

func TestYAMLCatalog_Update_Validation(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("rejects update with empty name", func(t *testing.T) {
		cat := newTestYAMLCatalog(t)
		p := catalog.NewProject("original", tempDir)
		require.NoError(t, cat.Add(p))

		p.Name = ""
		err := cat.Update(p)

		assert.ErrorIs(t, err, catalog.ErrEmptyName)
	})

	t.Run("rejects update with relative path", func(t *testing.T) {
		cat := newTestYAMLCatalog(t)
		p := catalog.NewProject("original", tempDir)
		require.NoError(t, cat.Add(p))

		p.Path = "relative/path"
		err := cat.Update(p)

		assert.ErrorIs(t, err, catalog.ErrRelativePath)
	})

	t.Run("rejects update with non-existent path", func(t *testing.T) {
		cat := newTestYAMLCatalog(t)
		p := catalog.NewProject("original", tempDir)
		require.NoError(t, cat.Add(p))

		p.Path = filepath.Join(tempDir, "nonexistent")
		err := cat.Update(p)

		assert.ErrorIs(t, err, catalog.ErrPathNotExist)
	})

	t.Run("allows updating to different valid path", func(t *testing.T) {
		cat := newTestYAMLCatalog(t)

		dir1 := filepath.Join(tempDir, "dir1")
		require.NoError(t, os.Mkdir(dir1, 0o755))

		dir2 := filepath.Join(tempDir, "dir2")
		require.NoError(t, os.Mkdir(dir2, 0o755))

		p := catalog.NewProject("myproject", dir1)
		require.NoError(t, cat.Add(p))

		p.Path = dir2
		err := cat.Update(p)

		assert.NoError(t, err)
	})
}

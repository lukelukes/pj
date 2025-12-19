package catalog_test

import (
	"os"
	"path/filepath"
	"pj/internal/catalog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

	t.Run("invalid status fails validation", func(t *testing.T) {
		p := catalog.NewProject("myproject", tempDir)
		p.Status = catalog.Status("invalid-status")

		err := p.ValidateAndNormalize()

		assert.ErrorIs(t, err, catalog.ErrInvalidStatus)
		assert.Contains(t, err.Error(), "invalid-status")
	})

	t.Run("valid status passes validation", func(t *testing.T) {
		testCases := []catalog.Status{
			catalog.StatusActive,
			catalog.StatusArchived,
			catalog.StatusAbandoned,
		}

		for _, status := range testCases {
			t.Run(string(status), func(t *testing.T) {
				p := catalog.NewProject("myproject", tempDir)
				p.Status = status

				err := p.ValidateAndNormalize()

				assert.NoError(t, err)
			})
		}
	})

	t.Run("empty status is allowed", func(t *testing.T) {
		p := catalog.NewProject("myproject", tempDir)
		p.Status = ""

		err := p.ValidateAndNormalize()

		assert.NoError(t, err)
	})

	t.Run("tag cleaning side effects", func(t *testing.T) {
		t.Run("validate cleans whitespace-only tags as side effect", func(t *testing.T) {
			p := catalog.NewProject("myproject", tempDir)
			p.Tags = []string{"valid", "  ", "\t\n", "another-valid", ""}

			err := p.ValidateAndNormalize()

			require.NoError(t, err)
			assert.Equal(t, []string{"valid", "another-valid"}, p.Tags)
		})

		t.Run("validate trims tag whitespace as side effect", func(t *testing.T) {
			p := catalog.NewProject("myproject", tempDir)
			p.Tags = []string{"  tag1  ", "\ttag2\n", " tag3 "}

			err := p.ValidateAndNormalize()

			require.NoError(t, err)
			assert.Equal(t, []string{"tag1", "tag2", "tag3"}, p.Tags)
		})

		t.Run("empty tag slice is preserved", func(t *testing.T) {
			p := catalog.NewProject("myproject", tempDir)
			p.Tags = []string{}

			err := p.ValidateAndNormalize()

			require.NoError(t, err)
			assert.Equal(t, []string{}, p.Tags)
		})

		t.Run("validate removes all-whitespace tags as side effect", func(t *testing.T) {
			p := catalog.NewProject("myproject", tempDir)
			p.Tags = []string{"  ", "\t", "\n", ""}

			err := p.ValidateAndNormalize()

			require.NoError(t, err)
			assert.Equal(t, []string{}, p.Tags)
		})

		t.Run("validate is idempotent", func(t *testing.T) {
			p := catalog.NewProject("myproject", tempDir)
			p.Tags = []string{"  tag1  ", "tag2", "  "}

			err := p.ValidateAndNormalize()
			require.NoError(t, err)
			tagsAfterFirst := make([]string, len(p.Tags))
			copy(tagsAfterFirst, p.Tags)

			err = p.ValidateAndNormalize()
			require.NoError(t, err)
			assert.Equal(t, tagsAfterFirst, p.Tags, "validate should be idempotent")
		})
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

	t.Run("rejects project with invalid status", func(t *testing.T) {
		cat := newTestYAMLCatalog(t)
		p := catalog.NewProject("myproject", tempDir)
		p.Status = catalog.Status("invalid")

		err := cat.Add(p)

		assert.ErrorIs(t, err, catalog.ErrInvalidStatus)
	})

	t.Run("cleans tags when adding", func(t *testing.T) {
		cat := newTestYAMLCatalog(t)
		p := catalog.NewProject("myproject", tempDir)
		p.Tags = []string{"valid", "  ", "another"}

		err := cat.Add(p)

		require.NoError(t, err)
		stored, _ := cat.Get(p.ID)
		assert.Equal(t, []string{"valid", "another"}, stored.Tags)
	})

	t.Run("accepts valid project", func(t *testing.T) {
		cat := newTestYAMLCatalog(t)
		p := catalog.NewProject("myproject", tempDir).
			WithTags("tag1", "tag2").
			WithStatus(catalog.StatusActive)

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

	t.Run("rejects update with invalid status", func(t *testing.T) {
		cat := newTestYAMLCatalog(t)
		p := catalog.NewProject("original", tempDir)
		require.NoError(t, cat.Add(p))

		p.Status = catalog.Status("invalid")
		err := cat.Update(p)

		assert.ErrorIs(t, err, catalog.ErrInvalidStatus)
	})

	t.Run("cleans tags when updating", func(t *testing.T) {
		cat := newTestYAMLCatalog(t)
		p := catalog.NewProject("myproject", tempDir)
		require.NoError(t, cat.Add(p))

		p.Tags = []string{"valid", "  ", "another"}
		err := cat.Update(p)

		require.NoError(t, err)
		stored, _ := cat.Get(p.ID)
		assert.Equal(t, []string{"valid", "another"}, stored.Tags)
	})

	t.Run("accepts valid update", func(t *testing.T) {
		cat := newTestYAMLCatalog(t)
		p := catalog.NewProject("original", tempDir)
		require.NoError(t, cat.Add(p))

		p.Notes = "Updated notes"
		p.Status = catalog.StatusArchived
		err := cat.Update(p)

		assert.NoError(t, err)
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

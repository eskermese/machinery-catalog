package media_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/eskermese/machinery-catalog/backend/internal/storage/media"
)

func TestStorage_Save_CreatesFileUnderRoot(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	storage := media.NewStorage(root)

	require.NoError(t, storage.Save(context.Background(), "machines/abc.jpg", []byte("картинка")))

	content, err := os.ReadFile(filepath.Join(root, "machines", "abc.jpg"))
	require.NoError(t, err)
	assert.Equal(t, "картинка", string(content))
}

// Путь за пределы корня отвергается, даже если его туда как-то занесло.
func TestStorage_Save_RejectsEscapingPath(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	storage := media.NewStorage(root)

	err := storage.Save(context.Background(), "../escaped.jpg", []byte("картинка"))
	require.Error(t, err)

	_, statErr := os.Stat(filepath.Join(filepath.Dir(root), "escaped.jpg"))
	assert.True(t, os.IsNotExist(statErr), "файл не должен был появиться за корнем")
}

func TestStorage_Delete_RemovesFile(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	storage := media.NewStorage(root)

	require.NoError(t, storage.Save(context.Background(), "machines/abc.jpg", []byte("картинка")))
	require.NoError(t, storage.Delete(context.Background(), "machines/abc.jpg"))

	_, err := os.Stat(filepath.Join(root, "machines", "abc.jpg"))
	assert.True(t, os.IsNotExist(err), "файл должен был исчезнуть")
}

// Удаление вызывается после того, как строка карточки уже исчезла из базы.
// Отсутствие файла на этом шаге — не отказ, а уже достигнутый результат.
func TestStorage_Delete_MissingFileIsNotAnError(t *testing.T) {
	t.Parallel()

	storage := media.NewStorage(t.TempDir())

	assert.NoError(t, storage.Delete(context.Background(), "machines/never-existed.jpg"))
}

// Путь приходит из базы, а не из домена, и удаление за корнем опаснее записи.
func TestStorage_Delete_RejectsEscapingPath(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	outside := filepath.Join(filepath.Dir(root), "outside.jpg")
	require.NoError(t, os.WriteFile(outside, []byte("чужой файл"), 0o600))

	err := media.NewStorage(root).Delete(context.Background(), "../outside.jpg")
	require.ErrorIs(t, err, media.ErrPathEscapesRoot)

	_, statErr := os.Stat(outside)
	assert.NoError(t, statErr, "чужой файл должен был уцелеть")
}

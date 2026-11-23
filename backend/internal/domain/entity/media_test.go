package entity_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/eskermese/machinery-catalog/backend/internal/domain/entity"
)

// jpegHeader, pngHeader, webpHeader — начала файлов настоящих форматов.
// Дальше идёт мусор: проверка смотрит только на сигнатуру.
var (
	jpegHeader = []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10}
	pngHeader  = []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A, 0x00}
	webpHeader = append([]byte("RIFF\x24\x00\x00\x00WEBP"), 0x00)
)

func TestNewStoredImage_DetectsTypeByContent(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		kind    entity.MediaKind
		content []byte
		suffix  string
		prefix  string
	}{
		{"jpeg в раздел машин", entity.MediaKindMachine, jpegHeader, ".jpg", "machines/"},
		{"png в раздел машин", entity.MediaKindMachine, pngHeader, ".png", "machines/"},
		{"webp в раздел программ", entity.MediaKindProgram, webpHeader, ".webp", "programs/"},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			stored, err := entity.NewStoredImage(testCase.kind, testCase.content)
			require.NoError(t, err)

			assert.True(t, strings.HasPrefix(stored.Path, testCase.prefix), stored.Path)
			assert.True(t, strings.HasSuffix(stored.Path, testCase.suffix), stored.Path)
		})
	}
}

// Расширение в имени берётся из содержимого: PDF, названный «.jpg», отвергается.
func TestNewStoredImage_RejectsUnsupportedContent(t *testing.T) {
	t.Parallel()

	cases := map[string][]byte{
		"pdf":          []byte("%PDF-1.7\n"),
		"текст":        []byte("совсем не картинка"),
		"gif":          []byte("GIF89a"),
		"обрезанный":   {0xFF},
		"riff не webp": []byte("RIFF\x24\x00\x00\x00AVI "),
	}

	for name, content := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := entity.NewStoredImage(entity.MediaKindMachine, content)
			require.ErrorIs(t, err, entity.ErrMediaTypeUnsupported)
		})
	}
}

func TestNewStoredImage_RejectsEmptyAndOversized(t *testing.T) {
	t.Parallel()

	_, err := entity.NewStoredImage(entity.MediaKindMachine, nil)
	require.ErrorIs(t, err, entity.ErrMediaFileMissing)

	oversized := append(bytes.Clone(jpegHeader), make([]byte, entity.MaxImageBytes)...)
	_, err = entity.NewStoredImage(entity.MediaKindMachine, oversized)
	require.ErrorIs(t, err, entity.ErrMediaTooLarge)
}

func TestNewStoredImage_RejectsUnknownKind(t *testing.T) {
	t.Parallel()

	_, err := entity.NewStoredImage(entity.MediaKind("../etc"), jpegHeader)
	require.ErrorIs(t, err, entity.ErrMediaKindUnknown)
}

// Имя генерируется сервером, поэтому два одинаковых файла не затирают друг друга.
func TestNewStoredImage_GeneratesUniqueNames(t *testing.T) {
	t.Parallel()

	first, err := entity.NewStoredImage(entity.MediaKindMachine, jpegHeader)
	require.NoError(t, err)

	second, err := entity.NewStoredImage(entity.MediaKindMachine, jpegHeader)
	require.NoError(t, err)

	assert.NotEqual(t, first.Path, second.Path)
}

func TestMediaKind_IsValid(t *testing.T) {
	t.Parallel()

	assert.True(t, entity.MediaKindMachine.IsValid())
	assert.True(t, entity.MediaKindProgram.IsValid())
	assert.False(t, entity.MediaKind("").IsValid())
	assert.False(t, entity.MediaKind("machines/../..").IsValid())
}

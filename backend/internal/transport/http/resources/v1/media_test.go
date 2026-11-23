package v1_test

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/eskermese/machinery-catalog/backend/internal/domain/entity"
	"github.com/eskermese/machinery-catalog/backend/internal/service"
	v1 "github.com/eskermese/machinery-catalog/backend/internal/transport/http/resources/v1"
)

// stubMediaService — заглушка сервиса изображений.
//
// Рукописная по той же причине, что и остальные заглушки транспорта:
// проверяется, ЧТО ушло в сервис — раздел и байты файла.
type stubMediaService struct {
	service.MediaService

	lastKind    entity.MediaKind
	lastContent []byte
	calls       int

	stored entity.StoredFile
	err    error
}

func (s *stubMediaService) SaveImage(
	_ context.Context,
	kind entity.MediaKind,
	content []byte,
) (entity.StoredFile, error) {
	s.calls++
	s.lastKind = kind
	s.lastContent = content

	return s.stored, s.err
}

// uploadRequest собирает multipart-запрос с одним файлом.
func uploadRequest(t *testing.T, target, field string, content []byte) *http.Request {
	t.Helper()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile(field, "клиентское-имя.jpg")
	require.NoError(t, err)

	_, err = part.Write(content)
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	request := httptest.NewRequest(http.MethodPost, target, body)
	request.Header.Set("Content-Type", writer.FormDataContentType())

	return request
}

// Раздел хранилища берётся из маршрута, а не из формы: клиент на него не влияет.
func TestMediaUpload_KindComesFromRoute(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		target string
		kind   entity.MediaKind
	}{
		"машины":           {"/machines", entity.MediaKindMachine},
		"программы":        {"/programs", entity.MediaKindProgram},
		"обложки разделов": {"/categories", entity.MediaKindCategory},
	}

	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			media := &stubMediaService{stored: entity.StoredFile{Path: "machines/abc.jpg"}}
			handler := v1.NewMediaAdminHandler(media).Routes()

			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, uploadRequest(t, testCase.target, "file", []byte{0xFF, 0xD8, 0xFF}))

			require.Equal(t, http.StatusCreated, recorder.Code)
			assert.Equal(t, testCase.kind, media.lastKind)
			assert.Equal(t, []byte{0xFF, 0xD8, 0xFF}, media.lastContent)
		})
	}
}

// Ответ содержит путь и адрес — ровно то, что клиент кладёт в форму карточки.
func TestMediaUpload_ReturnsPathAndURL(t *testing.T) {
	t.Parallel()

	media := &stubMediaService{stored: entity.StoredFile{Path: "machines/abc.jpg"}}
	handler := v1.NewMediaAdminHandler(media).Routes()

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, uploadRequest(t, "/machines", "file", []byte{0xFF, 0xD8, 0xFF}))

	require.Equal(t, http.StatusCreated, recorder.Code)

	var response struct {
		Path string `json:"path"`
		URL  string `json:"url"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))

	assert.Equal(t, "machines/abc.jpg", response.Path)
	assert.Equal(t, "/media/machines/abc.jpg", response.URL)
}

// Поле с другим именем — это запрос без файла, а не пятисотка.
func TestMediaUpload_MissingFileIs400(t *testing.T) {
	t.Parallel()

	media := &stubMediaService{}
	handler := v1.NewMediaAdminHandler(media).Routes()

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, uploadRequest(t, "/machines", "не-file", []byte{0xFF, 0xD8, 0xFF}))

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.Equal(t, 0, media.calls, "до сервиса запрос доходить не должен")
	assert.Contains(t, recorder.Body.String(), "SNT_MEDIA_FILE_MISSING")
}

// Тело сверх предела обрывается на чтении, а не дочитывается до сервиса.
//
// Проверяется именно транспортный предел: он чуть больше MaxImageBytes
// на служебную часть multipart, поэтому тело берётся с заведомым запасом.
// Файлы в промежутке между двумя пределами отвергает домен — см. entity.
func TestMediaUpload_OversizedBodyIs400(t *testing.T) {
	t.Parallel()

	media := &stubMediaService{}
	handler := v1.NewMediaAdminHandler(media).Routes()

	const overshoot = 1 << 20 // Мегабайт сверх предела: заведомо больше запаса на multipart

	oversized := append([]byte{0xFF, 0xD8, 0xFF}, make([]byte, entity.MaxImageBytes+overshoot)...)

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, uploadRequest(t, "/machines", "file", oversized))

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.Equal(t, 0, media.calls, "файл сверх предела не должен дочитываться до сервиса")
	assert.Contains(t, recorder.Body.String(), "SNT_MEDIA_TOO_LARGE")
}

// Отказ сервиса доезжает до клиента своим статусом и кодом.
func TestMediaUpload_ServiceErrorIsMapped(t *testing.T) {
	t.Parallel()

	media := &stubMediaService{err: entity.ErrMediaTypeUnsupported}
	handler := v1.NewMediaAdminHandler(media).Routes()

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, uploadRequest(t, "/machines", "file", []byte("%PDF")))

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "SNT_MEDIA_TYPE_UNSUPPORTED")
}

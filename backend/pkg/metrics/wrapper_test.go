package metrics

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStatusWriter_WriteHeaderAndWrite(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name        string
		headerCode  int      // 0 если WriteHeader не вызывать
		writes      []string // куски, которые будем писать через Write
		finalStatus int      // ожидаемый статус
		wantBody    string   // ожидаемое тело ответа
	}

	tests := []testCase{
		{
			name:        "Пустые значения",
			headerCode:  0,
			writes:      nil,
			finalStatus: 0,
			wantBody:    "",
		},
		{
			name:        "WriteHeader устанавливает кастомный статус",
			headerCode:  http.StatusTeapot, // 418
			writes:      nil,
			finalStatus: http.StatusTeapot,
			wantBody:    "",
		},
		{
			name:        "Write устанавливает статус 200, если WriteHeader не вызывался",
			headerCode:  0,
			writes:      []string{"hi"},
			finalStatus: http.StatusOK,
			wantBody:    "hi",
		},
		{
			name:        "Write не переопределяет заранее установленный статус",
			headerCode:  http.StatusAccepted, // 202 (вместо 204)
			writes:      []string{"ok"},
			finalStatus: http.StatusAccepted,
			wantBody:    "ok",
		},
		{
			name:        "Несколько вызовов Write сохраняют статус и дописывают тело",
			headerCode:  0,
			writes:      []string{"foo", "bar"},
			finalStatus: http.StatusOK,
			wantBody:    "foobar",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			sw := &statusWriter{ResponseWriter: rr}

			// вызывать WriteHeader если нужно
			if tt.headerCode != 0 {
				sw.WriteHeader(tt.headerCode)
			}

			// писать тело
			for _, chunk := range tt.writes {
				n, err := sw.Write([]byte(chunk))
				require.NoError(t, err)
				assert.Equal(t, len(chunk), n)
			}

			assert.Equal(t, tt.finalStatus, sw.status)
			// underlying recorder получает 200 по умолчанию
			if tt.finalStatus != 0 {
				assert.Equal(t, tt.finalStatus, rr.Code)
			}

			assert.Equal(t, tt.wantBody, rr.Body.String())
		})
	}
}

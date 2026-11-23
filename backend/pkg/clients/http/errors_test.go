package http

import (
	"bytes"
	"net/http"
	"testing"

	apperrors "github.com/eskermese/machinery-catalog/backend/pkg/apperrors"
	jsoniter "github.com/json-iterator/go"
	"github.com/ory/dockertest/v3/docker/pkg/ioutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExternalServiceError_Error(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name               string
		externalServiceErr apperrors.ExternalServiceError
		expErr             string
	}{
		{
			name: "Все поля заполнены",
			externalServiceErr: apperrors.ExternalServiceError{
				ServiceCode:    "HGL_INVALID_REQUEST",
				HTTPStatusCode: 400,
			},
			expErr: "http-статус: 400, код ошибки: 'HGL_INVALID_REQUEST'",
		},
		{
			name: "HTTP статус код не заполнен",
			externalServiceErr: apperrors.ExternalServiceError{
				ServiceCode: "HGL_INVALID_REQUEST",
			},
			expErr: "http-статус: 0, код ошибки: 'HGL_INVALID_REQUEST'",
		},
		{
			name: "Код сервиса не заполнен",
			externalServiceErr: apperrors.ExternalServiceError{
				HTTPStatusCode: 400,
			},
			expErr: "http-статус: 400, код ошибки: ''",
		},
	}

	for _, c := range cases {
		c := c

		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.expErr, c.externalServiceErr.Error())
		})
	}
}

func TestNewExternalServiceErrorFromHTTPResponse(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name               string
		respStatusCode     int
		respBody           string
		jsonAPI            jsoniter.API
		expExternalService *apperrors.ExternalServiceError
		expErr             string
	}{
		{
			name:           "Успешное декодирование",
			respStatusCode: 400,
			respBody:       `{"code":"HGL_INVALID_REQUEST"}`,
			jsonAPI:        jsoniter.ConfigFastest,
			expExternalService: &apperrors.ExternalServiceError{
				ServiceCode:    "HGL_INVALID_REQUEST",
				HTTPStatusCode: 400,
			},
			expErr: "",
		},
		{
			name:           "Ошибка декодирования",
			respStatusCode: 400,
			respBody:       `{"cod:"HGL_INVALID_REQUEST"}`,
			jsonAPI:        jsoniter.ConfigCompatibleWithStandardLibrary,
			expExternalService: &apperrors.ExternalServiceError{
				HTTPStatusCode: 400,
			},
			expErr: "не удалось распарсить тело ответа: apperrors.ExternalServiceError.readFieldHash: expect :, but found H, error found in #8 byte of ...|{\"cod:\"HGL_INVALID|..., bigger context ...|{\"cod:\"HGL_INVALID_REQUEST\"}|...; body: {\"cod:\"HGL_INVALID_REQUEST\"}",
		},
	}

	for _, c := range cases {
		c := c

		t.Run(c.name, func(t *testing.T) {
			resp := &http.Response{
				StatusCode: c.respStatusCode,
				Body:       ioutils.NewReadCloserWrapper(bytes.NewReader([]byte(c.respBody)), nil),
			}

			externalService, err := NewExternalServiceErrorFromHTTPResponse(resp)
			if c.expErr != "" {
				assert.EqualError(t, err, c.expErr)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, c.expExternalService, externalService)
		})
	}
}

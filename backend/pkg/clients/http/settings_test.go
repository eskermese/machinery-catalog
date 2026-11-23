package http

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestClientSettingsClient_GetClientSettings(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cases := []struct {
		name       string
		xLocation  string
		serverFunc http.HandlerFunc
		expRes     SettingsResponse
		expErr     string
	}{
		{
			name:      "Успешный запрос",
			xLocation: "123",
			serverFunc: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, "/settings/123", r.URL.Path)

				w.WriteHeader(http.StatusOK)
				fmt.Fprint(w, `{"lang": "en"}`)
			},
			expRes: SettingsResponse{
				Lang: "en",
			},
		},
		{
			name:      "Ошибка сервера",
			xLocation: "456",
			serverFunc: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprint(w, `{"code": "Internal error"}`)
			},
			expRes: SettingsResponse{},
			expErr: "http-статус: 500, код ошибки: 'Internal error'",
		},
		{
			name:      "Пустой ClientID",
			xLocation: "",
			serverFunc: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
				fmt.Fprint(w, `{"code": "Invalid client ID"}`)
			},
			expRes: SettingsResponse{},
			expErr: "http-статус: 400, код ошибки: 'Invalid client ID'",
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(tt.serverFunc)
			defer server.Close()

			client := NewSettingsClient(server.URL, "/settings/%s", time.Second)

			res, err := client.GetSettings(ctx, tt.xLocation)
			if tt.expErr != "" {
				assert.ErrorContains(t, err, tt.expErr)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.expRes, res)
		})
	}
}

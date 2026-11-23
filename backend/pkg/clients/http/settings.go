package http

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"resty.dev/v3"

	logger "github.com/eskermese/machinery-catalog/backend/pkg/logger"
)

// SettingsResponse ответ от сервиса настроек.
type SettingsResponse struct {
	Lang string `json:"lang"` // Язык приложения
}

// SettingsHTTPClient интерфейс для работы с настройками.
type SettingsHTTPClient interface {
	// GetSettings возвращает настройки.
	GetSettings(ctx context.Context, xLocation string) (SettingsResponse, error)
}

// settingsClient реализация для работы с настройками.
type settingsClient struct {
	client *resty.Client // HTTP клиент (Resty)

	getSettingsURI string // URI для получения настроек
}

// NewSettingsClient возвращает новый экземпляр Settings.
func NewSettingsClient(addr, getSettingsURI string, timeout time.Duration) SettingsHTTPClient {
	c := resty.New()
	c.SetTimeout(timeout)
	c.SetBaseURL(addr)
	c.AddRequestMiddleware(requestMiddleware)
	c.AddResponseMiddleware(responseMiddleware)

	return &settingsClient{
		client:         c,
		getSettingsURI: getSettingsURI,
	}
}

// GetSettings возвращает настройки.
func (c *settingsClient) GetSettings(ctx context.Context, xLocation string) (SettingsResponse, error) {
	path := fmt.Sprintf(c.getSettingsURI, xLocation)

	resp, err := c.client.R().
		SetContext(ctx).
		Get(path)
	if err != nil {
		return SettingsResponse{}, fmt.Errorf("выполнение запроса: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		raw := &http.Response{StatusCode: resp.StatusCode(), Body: io.NopCloser(bytes.NewReader(resp.Bytes()))}

		serviceCodeErr, perr := NewExternalServiceErrorFromHTTPResponse(raw)
		if perr != nil {
			logger.FromContext(ctx).WithFields(logger.Fields{
				"error": perr,
			}).Warn("ошибка при чтении тела ответа, возможно пустой ответ")
		}

		return SettingsResponse{}, serviceCodeErr
	}

	var settings SettingsResponse
	if err = json.Unmarshal(resp.Bytes(), &settings); err != nil {
		return SettingsResponse{}, fmt.Errorf("парсинг ответа: %w", err)
	}

	return settings, nil
}

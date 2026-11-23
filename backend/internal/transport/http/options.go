package http

import (
	"github.com/eskermese/machinery-catalog/backend/internal/service"
)

// Option определяет функцию для настройки HTTP сервера.
type Option func(*Server)

// WithAuthService устанавливает сервис выпуска токенов.
func WithAuthService(authService service.AuthService) Option {
	return func(srv *Server) {
		srv.authService = authService
	}
}

// WithCatalogService устанавливает сервис каталога.
func WithCatalogService(catalogService service.CatalogService) Option {
	return func(srv *Server) {
		srv.catalogService = catalogService
	}
}

// WithFXService устанавливает сервис курсов валют.
//
// Нужен публичной витрине: она показывает цену и в тенге. Отдельной опцией,
// как остальные сервисы, — сервер собирается из необязательных частей,
// и витрина без курса обязана подниматься.
func WithFXService(fxService service.FXService) Option {
	return func(srv *Server) {
		srv.fxService = fxService
	}
}

// WithSettingsService устанавливает сервис настроек показа цены.
func WithSettingsService(settingsService service.SettingsService) Option {
	return func(srv *Server) {
		srv.settingsService = settingsService
	}
}

// WithQuoteService устанавливает сервис коммерческих предложений.
func WithQuoteService(quoteService service.QuoteService) Option {
	return func(srv *Server) {
		srv.quoteService = quoteService
	}
}

// WithLeadService устанавливает сервис заявок.
func WithLeadService(leadService service.LeadService) Option {
	return func(srv *Server) {
		srv.leadService = leadService
	}
}

// WithTCOService устанавливает сервис стоимости владения.
func WithTCOService(tcoService service.TCOService) Option {
	return func(srv *Server) {
		srv.tcoService = tcoService
	}
}

// WithMediaService устанавливает сервис изображений.
func WithMediaService(mediaService service.MediaService) Option {
	return func(srv *Server) {
		srv.mediaService = mediaService
	}
}

// WithTokenParser устанавливает проверку токенов доступа.
func WithTokenParser(tokens tokenParser) Option {
	return func(srv *Server) {
		srv.tokens = tokens
	}
}

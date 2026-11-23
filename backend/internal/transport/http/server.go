package http

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"github.com/eskermese/machinery-catalog/backend/internal/config"
	"github.com/eskermese/machinery-catalog/backend/internal/domain/entity"
	"github.com/eskermese/machinery-catalog/backend/internal/domain/presenter"
	"github.com/eskermese/machinery-catalog/backend/internal/service"
	"github.com/eskermese/machinery-catalog/backend/internal/transport/http/resources"
	v1 "github.com/eskermese/machinery-catalog/backend/internal/transport/http/resources/v1"
	logger "github.com/eskermese/machinery-catalog/backend/pkg/logger"
	loggerMiddleware "github.com/eskermese/machinery-catalog/backend/pkg/logger/middleware"
	"github.com/eskermese/machinery-catalog/backend/pkg/metrics"
)

var (
	// ErrAuthServiceMissing сообщает, что серверу не передан сервис выпуска токенов.
	ErrAuthServiceMissing = errors.New("сервис выпуска токенов не задан")
	// ErrTokenParserMissing сообщает, что серверу не передана проверка токенов.
	ErrTokenParserMissing = errors.New("проверка токенов доступа не задана")
)

const (
	compressLevel = 5                // Уровень сжатия.
	readTimeout   = 5 * time.Second  // Время ожидания запроса.
	writeTimeout  = 30 * time.Second // Время ожидания ответа.
	maxAge        = 300              // Время жизни C.O.R.S. заголовков.
)

// Server представляет собой HTTP сервер.
type Server struct {
	Address     string             // Адрес сервера
	BasePath    string             // Базовый путь
	FilesDir    string             // Директория с файлами
	MediaDir    string             // Каталог с изображениями каталога техники
	Environment config.Environment // Окружение

	serviceName     string        // Название сервиса
	idleConnsClosed chan struct{} // Способ определить незавершенные соединения
	version         string        // Версия приложения
	gatewayToken    string        // Общий секрет доверенного шлюза (отладочный режим)
	headerAuth      bool          // Разрешён отладочный вход по заголовкам

	authService     service.AuthService     // Сервис выпуска токенов
	catalogService  service.CatalogService  // Сервис каталога
	fxService       service.FXService       // Сервис курсов валют
	settingsService service.SettingsService // Сервис настроек показа цены
	quoteService    service.QuoteService    // Сервис коммерческих предложений
	leadService     service.LeadService     // Сервис заявок
	tcoService      service.TCOService      // Сервис стоимости владения
	mediaService    service.MediaService    // Сервис изображений
	tokens          tokenParser             // Проверка токенов доступа

	baseCurrency  string // Валюта прайса
	quoteCurrency string // Валюта, в которую цена пересчитывается обязательно
	usdCurrency   string // Валюта второго отображения цены
}

// tokenParser проверяет токен доступа. Объявлен здесь, чтобы сервер не зависел
// от конкретной реализации выпуска токенов.
type tokenParser interface {
	Parse(token string) (entity.Actor, error)
}

// NewServer создает новый HTTP сервер.
func NewServer(cfg *config.Config, options ...Option) *Server {
	srv := &Server{
		Address:     cfg.GetHTTPDomain(),
		FilesDir:    cfg.FilesDir,
		MediaDir:    cfg.MediaDir,
		BasePath:    cfg.BasePath,
		Environment: cfg.Environment,

		idleConnsClosed: make(chan struct{}),
		serviceName:     cfg.ServiceName,
		version:         cfg.Version,
		gatewayToken:    cfg.GatewayToken,
		headerAuth:      cfg.IsHeaderAuthAllowed(),

		// Валюты — общие константы домена, а не настройка и не параметр
		// запроса. Настройкой они разошлись бы с парой, в которой считаются
		// предложение и стоимость владения: витрина показывала бы цену
		// по одному курсу, а КП выпускалось бы по другому.
		baseCurrency:  entity.CurrencyCNY,
		quoteCurrency: entity.CurrencyKZT,
		usdCurrency:   entity.CurrencyUSD,
	}

	for _, opt := range options {
		opt(srv)
	}

	return srv
}

// setupRouter инициализирует HTTP роутер. Функция используется для подключения middleware и маппинга ресурсов.
func (srv *Server) setupRouter() chi.Router {
	r := chi.NewRouter()

	r.Group(func(r chi.Router) {
		// монтируем дополнительные ресурсы
		r.Handle("/metrics", promhttp.Handler())
		r.Mount("/version", resources.VersionResource{Version: srv.version}.Routes())
	})

	r.Group(func(r chi.Router) {
		r.Use(metrics.NewMiddlewareMetric(srv.serviceName).Middleware) // собирает метрики по каждому запросу
		r.Use(middleware.NoCache)                                      // no-cache
		r.Use(middleware.RequestID)                                    // вставляет request ID в контекст каждого запроса
		r.Use(loggerMiddleware.Logger(nil))                            // логирует начало и окончание каждого запроса с указанием времени обработки
		r.Use(middleware.Recoverer)                                    // управляемо обрабатывает паники и выдает stack trace при их возникновении
		r.Use(middleware.RealIP)                                       // устанавливает RemoteAddr для каждого запроса с заголовками X-Forwarded-For или X-Real-IP
		r.Use(middleware.NewCompressor(compressLevel).Handler)
		r.Use(cors.Handler(cors.Options{
			AllowedOrigins:   allowedOrigins(srv.Environment),
			AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
			AllowedHeaders:   []string{"*"},
			ExposedHeaders:   []string{"Link"},
			AllowCredentials: false,
			MaxAge:           maxAge, // Максимальное время жизни C.O.R.S. заголовков.
		}))

		// Выпуск токенов идёт вне аутентификации: получить токен, имея токен, незачем.
		r.Mount("/api/v1/auth", v1.NewAuthHandler(srv.authService, srv.authMiddleware()).Routes())

		// Публичная витрина каталога идёт без аутентификации: каталог читают
		// до входа. Обработчик держит только публичные методы сервиса,
		// поэтому черновик отсюда не отдаётся ни при каких параметрах.
		r.Mount("/api/v1/catalog", v1.NewCatalogHandler(
			srv.catalogService, srv.fxService, srv.settingsService,
			srv.baseCurrency, srv.quoteCurrency, srv.usdCurrency,
		).Routes())

		// Страница предложения по публичной ссылке, справочник программ лизинга
		// и приём заявок. Токен предложения — единственная защита страницы,
		// поэтому он не попадает ни в лог запросов, ни в метрики: и то и другое
		// маркирует запрос шаблоном маршрута, а не конкретным путём.
		r.Mount("/api/v1", v1.NewQuoteHandler(
			srv.quoteService, srv.leadService, srv.tcoService, srv.catalogService).Routes())

		// Изображения каталога раздаёт сам сервис, без аутентификации:
		// фотографии техники — часть публичной витрины. Маршрут смонтирован
		// во всех окружениях, в отличие от /files: адрес фотографии обязан
		// работать там же, где работает каталог.
		r.Mount(presenter.MediaRoutePrefix, resources.FilesResource{FilesDir: srv.MediaDir}.Routes())

		// Прикладное API закрыто аутентификацией. Сервисные маршруты
		// (/metrics, /version, /files, /swagger) остаются снаружи:
		// их дергает инфраструктура, а не пользователь.
		r.Group(func(r chi.Router) {
			r.Use(srv.authMiddleware())

			// Админка каталога доступна менеджеру и администратору; разделение
			// прав внутри раздела делает сам обработчик — см. его Routes.
			r.Group(func(r chi.Router) {
				r.Use(v1.RoleAnyMiddleware(entity.ActorTypeManager, entity.ActorTypeAdmin))

				r.Mount("/api/v1/admin/catalog", v1.NewCatalogAdminHandler(srv.catalogService).Routes())

				// Загрузка изображений вынесена отдельным разделом: она нужна
				// и карточкам каталога, и логотипам программ лизинга, то есть
				// не принадлежит ни одному из них.
				r.Mount("/api/v1/admin/media", v1.NewMediaAdminHandler(srv.mediaService).Routes())

				r.Mount("/api/v1/admin",
					v1.NewQuoteAdminHandler(srv.quoteService, srv.leadService).Routes())

				// Курсы, наценки и настройки показа цены. Права внутри
				// разделены по цене ошибки, а не по должности — см. Routes.
				r.Mount("/api/v1/admin/fx", v1.NewFXAdminHandler(
					srv.fxService, srv.settingsService,
					srv.baseCurrency, srv.quoteCurrency, srv.usdCurrency,
				).Routes())
			})
		})

		if !srv.Environment.IsProduction() {
			r.Mount("/files", resources.FilesResource{FilesDir: srv.FilesDir}.Routes())
			r.Mount("/swagger", resources.SwaggerResource{FilesPath: "/files", BasePath: srv.BasePath}.Routes())
		}
	})

	return r
}

// authMiddleware выбирает схему аутентификации.
//
// По умолчанию — токен. Отладочная схема по заголовкам подключается к роутеру
// только в окружении local и только при явно выставленном DEBUG_HEADER_AUTH:
// вне local этой ветки в цепочке обработки нет вовсе, поэтому неверно
// выставленный флаг в проде ничего не открывает.
func (srv *Server) authMiddleware() func(http.Handler) http.Handler {
	if srv.headerAuth {
		gateway := v1.GatewayMiddleware(srv.gatewayToken, srv.Environment.IsLocal())
		headers := v1.DebugHeaderAuthMiddleware()

		return func(next http.Handler) http.Handler {
			return gateway(headers(next))
		}
	}

	return v1.AuthMiddleware(srv.tokens)
}

// getAllowedOrigins возвращает список хостов для C.O.R.S.
func allowedOrigins(environment config.Environment) []string {
	if environment.IsProduction() {
		return []string{"*"}
	}

	return []string{"*"}
}

// validate проверяет, что сервер собран пригодным к работе.
//
// Проверка нужна потому, что зависимости приходят опциями: забытая опция раньше
// проявлялась паникой на первом же запросе к защищённому маршруту. Отказ на
// старте честнее — по той же причине, по которой сервис не поднимается без
// JWT_SECRET вне local.
func (srv *Server) validate() error {
	if srv.authService == nil {
		return ErrAuthServiceMissing
	}

	// В отладочном режиме токены не разбираются вовсе, поэтому парсер там не нужен.
	if srv.tokens == nil && !srv.headerAuth {
		return ErrTokenParserMissing
	}

	return nil
}

// Run запускает HTTP или HTTPS листенер в зависимости от того как заполнена
// структура Server{}.
func (srv *Server) Run(ctx context.Context) error {
	if err := srv.validate(); err != nil {
		return err
	}

	s := http.Server{
		Addr:         srv.Address,
		Handler:      h2c.NewHandler(srv.setupRouter(), &http2.Server{}), // Подключаем HTTP/2 поддержку без TLS
		ReadTimeout:  readTimeout,                                        // wait() + tls handshake + req.headers + req.body
		WriteTimeout: writeTimeout,                                       // все что выше + response
	}

	log := logger.FromContext(ctx)

	go func() {
		<-ctx.Done()

		const timeout = 5 * time.Second

		sCtx, cancel := context.WithTimeout(context.TODO(), timeout)
		defer cancel()

		defer close(srv.idleConnsClosed)

		log.Info("HTTP сервер остановлен")

		if err := s.Shutdown(sCtx); err != nil {
			log.WithFields(logger.Fields{
				"error": err,
			}).Warn("HTTP сервер не остановлен")
		}
	}()

	log.WithFields(logger.Fields{
		"address": srv.Address,
	}).Infof("HTTP сервер запущен")

	if err := s.ListenAndServe(); err != nil {
		srv.Wait()

		return err
	}

	return nil
}

// Wait ожидает момента завершения обработки всех соединений.
func (srv *Server) Wait() {
	<-srv.idleConnsClosed
}

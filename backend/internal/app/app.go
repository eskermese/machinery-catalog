package app

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/grafana/pyroscope-go"
	"golang.org/x/sync/errgroup"

	"github.com/eskermese/machinery-catalog/backend/internal/auth"
	"github.com/eskermese/machinery-catalog/backend/internal/client/nationalbank"
	"github.com/eskermese/machinery-catalog/backend/internal/config"
	"github.com/eskermese/machinery-catalog/backend/internal/domain/entity"
	"github.com/eskermese/machinery-catalog/backend/internal/service"
	"github.com/eskermese/machinery-catalog/backend/internal/storage/dragonfly"
	mediastorage "github.com/eskermese/machinery-catalog/backend/internal/storage/media"
	"github.com/eskermese/machinery-catalog/backend/internal/storage/postgres"
	"github.com/eskermese/machinery-catalog/backend/internal/transport/http"
	"github.com/eskermese/machinery-catalog/backend/internal/worker"
	logger "github.com/eskermese/machinery-catalog/backend/pkg/logger"
)

const (
	fileVersion    = "version" // Файл с версией приложения
	defaultVersion = "unknown" // Версия по умолчанию
)

// gracefulShutdownTimeout представляет собою время, за которое должно сработать корректное завершение программы.
const gracefulShutdownTimeout = 5 * time.Second

// Run запускает приложение.
func Run(cfg *config.Config) error {
	// Инициализация контекста для корректного завершения программы.
	// При получении сигнала SIGTERM или SIGINT, будет вызвана функция cancel.
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer cancel()

	cfg.Version = readVersion(fileVersion)

	// Инициализация логгера.
	log := logger.NewZeroLogger(cfg.LogLevel, cfg.ServiceName, cfg.Version)
	logger.DefaultLoggerInContext = log

	log.WithFields(logger.Fields{
		"version":      cfg.Version,
		"database_url": cfg.DSURL,
		"cache_addr":   cfg.CacheAddr,
	}).Info("Запуск приложения...")

	// Инициализация Pyroscope профайлера.
	if cfg.IsPyroscopeEnabled() {
		if _, err := pyroscope.Start(pyroscope.Config{
			ApplicationName: cfg.ServiceName,
			ServerAddress:   cfg.PyroscopeAddr,
			Tags: map[string]string{
				"env":     cfg.Environment.Name,
				"version": cfg.Version,
			},
			ProfileTypes: []pyroscope.ProfileType{
				pyroscope.ProfileCPU,           // CPU-профилирование
				pyroscope.ProfileGoroutines,    // Горутины
				pyroscope.ProfileAllocObjects,  // Выделенные объекты
				pyroscope.ProfileInuseObjects,  // Используемые объекты
				pyroscope.ProfileMutexCount,    // Счётчик мьютексов
				pyroscope.ProfileMutexDuration, // Длительность мьютексов
			},
		}); err != nil {
			log.Warnf("Ошибка инициализации профайлера: %s", err)
		}
	}

	// Инициализация базы данных.
	ds := postgres.New(&cfg.Database)

	if err := ds.Connect(); err != nil {
		return fmt.Errorf("подключение к базе данных: %w", err)
	}

	defer func() {
		// Даем таймаут корректному завершению программы.
		// Если через определенное время соединение не закроется, то закрыть принудительно.
		sCtx, sCancel := context.WithTimeout(context.Background(), gracefulShutdownTimeout)
		defer sCancel()

		if cErr := ds.Close(sCtx); cErr != nil {
			log.WithFields(logger.Fields{
				"error": cErr,
			}).Warn("закрытие базы данных")
		}
	}()

	log.Info("Подключение к базе данных успешно")

	// Инициализация кэша. Недоступность кэша не должна валить старт: прикладных
	// сценариев поверх него пока нет, а проверка на старте делает проблему видной
	// сразу, а не на первом обращении.
	cacheData := dragonfly.New(cfg.Cache)
	if err := cacheData.Ping(); err != nil {
		log.WithFields(logger.Fields{
			"error": err,
		}).Warn("кэш недоступен")
	}

	// Токены доступа. Пустой секрет вне local уже отсеян валидацией конфигурации,
	// в local сервис поднимается, но выпуск токенов будет отвечать ошибкой.
	tokenService := auth.NewTokenService(cfg.JWTSecret, auth.TokenTTL)
	if !tokenService.IsConfigured() {
		log.Warn("JWT_SECRET не задан: выпуск и проверка токенов недоступны")
	}

	// Инициализация сервисов.
	authService := service.NewAuthService(ds.UserRepository(), tokenService, cfg.Auth.SelfRegistration)

	// Источник курса — официальная выдача Нацбанка Казахстана.
	//
	// Сам по себе он ничего не загружает: курс тянет воркер синхронизации,
	// и включается тот отдельно, флагом FX_SYNC_ENABLED. Поэтому подключённый
	// источник и работающая синхронизация — разные вещи, и одно без другого
	// не означает, что курс в базе свежий.
	fxService := service.NewFXService(
		ds.FXRateRepository(), ds.FXMarkupRepository(), cacheData.FXCache(),
		nationalbank.New(cfg.FXProviderURL, cfg.FXProviderTimeout),
		entity.CurrencyKZT,
		cfg.FXRateStaleAfterDays,
	)

	// Изображения лежат в каталоге на диске и раздаются самим сервисом:
	// объектного хранилища в развёртывании нет, а порт позволит его подключить,
	// не трогая ни домен, ни сервис.
	//
	// Хранилище одно на двоих: сервис изображений в него пишет, каталог —
	// убирает файлы удалённых карточек. Два экземпляра поверх одного каталога
	// работали бы так же, но заводить их значило бы делать вид, что хранилищ
	// два.
	imageStorage := mediastorage.NewStorage(cfg.MediaDir)

	catalogService := service.NewCatalogService(
		ds.CategoryRepository(), ds.AttributeRepository(), ds.MachineRepository(),
		ds.MachineAttributeValueRepository(), ds.MachineImageRepository(),
		imageStorage, cacheData.CatalogCache(), ds.ResidualValueRepository(), ds.TxManager(),
	)

	// Коммерческие предложения и заявки.
	quoteService := service.NewQuoteService(
		ds.QuoteRepository(), ds.LeasingProgramRepository(), catalogService, fxService,
	)
	leadService := service.NewLeadService(ds.LeadRepository(), ds.Notifier())

	// Стоимость владения считается по карточке, кривой остаточной стоимости
	// раздела и текущему курсу: цена машины хранится в долларах, а топливо
	// и оператор приходят от клиента в тенге.
	tcoService := service.NewTCOService(catalogService, ds.ResidualValueRepository(), fxService)

	// Настройки показа цены. Лежат в базе, а не в переменных окружения:
	// включатель доллара переключает администратор из интерфейса, а
	// переменная означала бы перезапуск сервиса на каждое переключение.
	settingsService := service.NewSettingsService(ds.AppSettingsRepository())

	mediaService := service.NewMediaService(imageStorage)

	// Доставка уведомлений. Канал пока не подключён: заглушка отказывает явной
	// ошибкой, поэтому события копятся в очереди с пустым processed_at,
	// а незаконченная настройка видна и в логах, и в базе.
	notificationService := service.NewNotificationService(
		ds.OutboxRepository(), service.NewNotConfiguredSender(),
	)

	// Администраторы заводятся заранее: самостоятельная регистрация администратора
	// означала бы, что права раздаёт тот, кто их запрашивает.
	if err := authService.EnsureAdmins(ctx, cfg.AdminPhoneList(), time.Now().UTC()); err != nil {
		return fmt.Errorf("посадка администраторов: %w", err)
	}

	g, gCtx := errgroup.WithContext(ctx)

	// HTTP Сервер.
	g.Go(func() error {
		httpOpts := []http.Option{
			http.WithAuthService(authService),
			http.WithTokenParser(tokenService),
			http.WithCatalogService(catalogService),
			http.WithFXService(fxService),
			http.WithSettingsService(settingsService),
			http.WithQuoteService(quoteService),
			http.WithLeadService(leadService),
			http.WithTCOService(tcoService),
			http.WithMediaService(mediaService),
		}

		httpServer := http.NewServer(cfg, httpOpts...)

		return httpServer.Run(gCtx)
	})

	// Фоновая синхронизация курса. Выключена по умолчанию: пока источник
	// не подключён, каждый её прогон только пишет предупреждение в лог.
	if cfg.IsFXSyncEnabled() {
		fxSync := worker.NewFXSync(fxService, cfg.FXSyncInterval, entity.CurrencyCNY, entity.CurrencyKZT)

		g.Go(func() error {
			return fxSync.Run(gCtx)
		})
	}

	// Отправка уведомлений из очереди. Выключается тем же флагом, что
	// и синхронизация курса: пока внешние каналы не подключены, оба воркера
	// только пишут предупреждения в лог.
	if cfg.IsOutboxSyncEnabled() {
		outboxSync := worker.NewOutboxSync(notificationService, cfg.OutboxSyncInterval, cfg.OutboxBatchSize)

		g.Go(func() error {
			return outboxSync.Run(gCtx)
		})
	}

	if err := g.Wait(); err != nil {
		return fmt.Errorf("работа основных горутин: %w", err)
	}

	return nil
}

// readVersion читает версию приложения.
func readVersion(filePath string) string {
	file, err := os.Open(filePath)
	if err != nil {
		return defaultVersion
	}
	defer file.Close()

	var version string
	if _, err = fmt.Fscanln(file, &version); err != nil {
		return defaultVersion
	}

	return version
}

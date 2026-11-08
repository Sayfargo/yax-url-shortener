// Модуль app отвечает за оркестрацию, сборку и жизненный цикл всего приложения.
// Он инициализирует зависимости, настраивает маршрутизацию, управляет
// переключением репозиториев и обеспечивает graceful shutdown.
package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	_ "net/http/pprof"

	"github.com/Sayfargo/yax-url-shortener/internal/audit"
	"github.com/Sayfargo/yax-url-shortener/internal/config"
	"github.com/Sayfargo/yax-url-shortener/internal/core/cache"
	"github.com/Sayfargo/yax-url-shortener/internal/core/db/postgres"
	"github.com/Sayfargo/yax-url-shortener/internal/core/filestorage"
	"github.com/Sayfargo/yax-url-shortener/internal/core/httpserver"
	"github.com/Sayfargo/yax-url-shortener/internal/core/transport/http/middleware"
	"github.com/Sayfargo/yax-url-shortener/internal/handler"
	"github.com/Sayfargo/yax-url-shortener/internal/repository/url"
	"github.com/Sayfargo/yax-url-shortener/internal/service"
	"github.com/Sayfargo/yax-url-shortener/pkg/generator"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

// App инкапсулирует в себе все ключевые компоненты запущенного приложения:
// HTTP-сервер, пулы соединений и слой бизнес-логики.
type App struct {
	// Server управляет жизненным циклом HTTP-сервера.
	server *httpserver.HTTPServer
	// DB хранит пул соединений с PostgreSQL. Может быть nil, если выбран другой тип хранилища.
	db *pgxpool.Pool
	// Storage управляет дескриптором файла для сохранения кэша на диск. Может быть nil.
	storage *filestorage.FileStorage
	// Service представляет собой слой ключевой бизнес-логики по сокращению URL.
	svc          *service.URLShortenerService
	auditFileSub *audit.FileSubscriber
	auditSvc     *audit.AuditService
}

const (
	defaultSecretKey = "0123456789abcdef0123456789abcdef"
	alphabet         = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	size             = 8
)

// New создает, связывает зависимости и инициализирует новый экземпляр [App].
//
// Функция выполняет следующие шаги:
// 1. Инициализирует middleware шифрования (куки/авторизация), считывая ключ из CK_KEY.
// 2. Настраивает глобальный роутер [chi.Mux], добавляя логирование, сжатие и эндпоинты pprof (/debug).
// 3. Динамически определяет и поднимает нужный тип репозитория (DB, File, In-Memory) на основе [config.Config].
// 4. Настраивает подписки для модуля аудита (в файл и/или по HTTP).
// 5. Инициализирует генератор идентификаторов, сервисы и хэндлеры API.
//
// Возвращает инициализированный экземпляр [App] или ошибку, если инициализация любого из компонентов сорвалась.
func New(cfg *config.Config, log *slog.Logger) (*App, error) {

	app := &App{}

	// secret key
	secretKey := os.Getenv("CK_KEY")

	if secretKey == "" {
		secretKey = defaultSecretKey
	}

	// root router
	rootRouter := chi.NewRouter()
	rootRouter.Use(middleware.Logging(log))
	rootRouter.Use(middleware.GzipCompress())

	// pprof
	rootRouter.Mount("/debug", chimiddleware.Profiler())

	var (
		db          *pgxpool.Pool
		fileStorage *filestorage.FileStorage
		activeRepo  service.URLRepository
		err         error
	)

	switch cfg.StorageType() {
	case config.StorageTypeDB:
		db, err = postgres.New(*cfg.DB, log)
		if err != nil {
			log.Error("failed to create db connection pool", "err", err)
			return nil, fmt.Errorf("database initialize: %w", err)
		}
		activeRepo = url.NewPgRepo(db)

	case config.StorageTypeFile:
		fileStorage, err = filestorage.Init(cfg.FileStorage)
		if err != nil {
			log.Error("failed to initialize file storage", "err", err)
			return nil, fmt.Errorf("file storage init: %w", err)
		}

		log.Info(
			"initialized file storage with cache repository",
			"file_storage_path", cfg.FileStorage.FilePath,
		)

		cacheStorage := cache.Init()
		cr := url.NewInMemoryRepo(cacheStorage)

		fcr, err := url.NewFileInMemoryRepository(cr, fileStorage)
		if err != nil {
			log.Error("failed to initialize file cache repository", "err", err)
			_ = fileStorage.Close()
			return nil, fmt.Errorf("restore cache: %w", err)
		}

		activeRepo = fcr

	case config.StorageTypeMemory:

		log.Info(
			"initialized only cache repository",
		)

		cacheStorage := cache.Init()
		activeRepo = url.NewInMemoryRepo(cacheStorage)
	}

	// Audit service initialization
	publisher := audit.NewPublisher()

	if cfg.Audit.File != "" {
		fileSub, err := audit.NewFileSubscriber(cfg.Audit.File)
		if err != nil {
			log.Error("failed to initialize file subscriber for audit", "err", err)
			return nil, fmt.Errorf("audit file subscriber init: %w", err)
		}
		publisher.Subscribe(fileSub)
		app.auditFileSub = fileSub
	}

	if cfg.Audit.URL != "" {
		httpSub := audit.NewHTTPSubscriber(cfg.Audit.URL)
		publisher.Subscribe(httpSub)
	}

	auditService := audit.NewAuditService(publisher, log, 1024, 16, 15*time.Second)

	// id generator

	generator, err := generator.New(alphabet, size)
	if err != nil {
		log.Error("failed to initialize generator", "err", err)
		return nil, fmt.Errorf("id generator init: %w", err)
	}

	svc := service.New(activeRepo, generator, cfg.Server.BaseURL, log)
	h := handler.New(svc, log, db, auditService)
	h.RegisterPublicAPI(rootRouter)
	rootRouter.Group(func(r chi.Router) {
		r.Use(middleware.Auth(string(secretKey)))

		h.RegisterPrivateAPI(r)
	})

	httpServer := httpserver.New(rootRouter, cfg.Server, log)

	app.server = httpServer
	app.db = db
	app.storage = fileStorage
	app.svc = svc
	app.auditSvc = auditService

	return app, nil

}

// Run запускает асинхронные воркеры очистки и разворачивает основной HTTP-сервер.
// Метод блокирует текущую горутину до тех пор, пока контекст [context.Context] не будет отменен.
//
// Перед завершением метод гарантирует безопасное закрытие всех активных ресурсов:
// файлового хранилища (сброс кэша) и пула базы данных PostgreSQL через [errors.Join].
func (a *App) Run(ctx context.Context) (errs error) {

	defer func() {

		if a.storage != nil {
			if err := a.storage.Close(); err != nil {
				errs = errors.Join(errs, fmt.Errorf("file storage close: %w", err))
			}
		}

		if a.auditFileSub != nil {
			if err := a.auditFileSub.Close(); err != nil {
				errs = errors.Join(errs, fmt.Errorf("audit file sub close: %w", err))
			}
		}

		if a.db != nil {
			a.db.Close()
		}
	}()

	a.svc.StartDeleteWorker(ctx)
	a.auditSvc.StartAuditWorkerPool(ctx)

	if err := a.server.Run(ctx); err != nil {
		errs = errors.Join(errs, fmt.Errorf("server run: %w", err))
	}

	if errs != nil {
		return errs
	}

	return nil
}

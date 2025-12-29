package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"main_service/internal/config"
	addProduct "main_service/internal/http-server/handlers/products/add"
	deleteProduct "main_service/internal/http-server/handlers/products/delete"
	getProducts "main_service/internal/http-server/handlers/products/get"
	getByID "main_service/internal/http-server/handlers/products/get_by_id"
	"main_service/internal/lib/jwt"
	"main_service/internal/lib/parser"
	authMiddlware "main_service/internal/middleware/auth"
	"main_service/internal/middleware/products"
	"main_service/internal/rabbitmq"
	"main_service/internal/storage/postgres"
	"main_service/internal/storage/redis"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-playground/validator/v10"
)

const (
	envLocal = "local"
	envDev   = "dev"
	envProd  = "prod"
)

func main() {
	cfg := config.MustLoad("./config/config.yaml")

	log := setupLogger(cfg.Env)

	log.Info("starting main service", slog.String("env", cfg.Env))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		log.Info("Shutdown signal received")
		cancel()
	}()

	jwtParser := jwt.New(cfg.JWTSecret)

	// * Инициализация Redis
	redisClient, err := redis.New(ctx, cfg.Redis.Addr, cfg.Redis.Db, cfg.Redis.DefaultTTL)
	if err != nil {
		log.Error("failed to connect redis", slog.String("err", err.Error()))
		os.Exit(1)
	}
	defer redisClient.Close()

	// * Инициализация PosgreSQL
	postgresClient, err := postgres.New(ctx, cfg)
	if err != nil {
		log.Error("failed to connect posgtreSQL", slog.String("err", err.Error()))
		os.Exit(1)
	}
	defer postgresClient.Close()

	// * Инициализация RabbitMQ
	rabbitMQClient, err := rabbitmq.New(cfg.RabbitMQ.URL)
	if err != nil {
		log.Error("failed to connect rabbitMQ", slog.String("err", err.Error()))
		os.Exit(1)
	}
	defer rabbitMQClient.Close()

	rabbitMQProducer := rabbitmq.NewProducer(
		rabbitMQClient.Channel,
		cfg.RabbitMQ.QueueName,
	)
	rabbitMQConsumer := rabbitmq.NewConsumer(
		rabbitMQClient.Channel,
		log,
		cfg.RabbitMQ.QueueName,
		cfg.RabbitMQ.WorkerPoolSize,
	)

	// * Инициализация Products Middleware
	prodOP := products.New(
		postgresClient,
		redisClient,
		rabbitMQProducer,
		cfg.CheckInterval,
	)

	// * Инициализация parser'а
	parserClient := parser.New(postgresClient, rabbitMQConsumer)

	requestValidator := validator.New()

	router := setupRouter(
		ctx,
		log,
		requestValidator,
		postgresClient,
		*prodOP,
		*jwtParser,
	)
}

func setupRouter(
	ctx context.Context,
	log *slog.Logger,
	validate *validator.Validate,
	postgres *postgres.PostgresRepo,
	prodOP products.ProductOperator,
	jwtParser jwt.JWTParser,
) *chi.Mux {
	r := chi.NewRouter()

	r.Use(authMiddlware.New(log, jwtParser))
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Post("/product", addProduct.New(ctx, log, prodOP, jwtParser, validate))
	r.Get("/products", getProducts.New(ctx, log, jwtParser, postgres))
	r.Get("/product", getByID.New(ctx, log, &prodOP, jwtParser))
	r.Delete("/product", deleteProduct.New(ctx, log, postgres, jwtParser))

	return r
}

func setupLogger(env string) *slog.Logger {
	var log *slog.Logger

	switch env {
	case envLocal:
		log = slog.New(
			slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}),
		)
	case envDev:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}),
		)
	case envProd:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}),
		)
	}

	return log
}

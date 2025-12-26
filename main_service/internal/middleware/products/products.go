package products

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"main_service/internal/models"
	"main_service/internal/storage"
)

type RedisStorage interface {
	SaveProduct(ctx context.Context, product models.Product) error
	Product(ctx context.Context, productID int64) (models.Product, error)
}

type PostgresStorage interface {
	SaveProduct(ctx context.Context, userID int64, productURL, title string) (int64, error)
	ProductByID(ctx context.Context, productID int64) (models.Product, error)
}

type RabbitMQ interface {
	PublishJSON(ctx context.Context, msg any) error
}

type ProductOperator struct {
	CheckInterval time.Duration
	Redis         RedisStorage
	Postgres      PostgresStorage
	Rabbitmq      RabbitMQ
}

func New(p PostgresStorage, r RedisStorage, rabbit RabbitMQ, checkInterval time.Duration) *ProductOperator {
	return &ProductOperator{
		CheckInterval: checkInterval,
		Redis:         r,
		Postgres:      p,
		Rabbitmq:      rabbit,
	}
}

func (p *ProductOperator) SaveProduct(ctx context.Context, url, title string, userID int64) (int64, error) {
	productID, err := p.Postgres.SaveProduct(ctx, userID, url, title)
	if err != nil {
		return 0, err
	}

	product := models.ProductForProducer{
		ID:  productID,
		URL: url,
	}

	data, err := json.Marshal(product)
	if err != nil {
		return 0, fmt.Errorf("failed to serialize product: %w", err)
	}

	err = p.Rabbitmq.PublishJSON(ctx, data)
	if err != nil {
		return 0, err
	}

	return productID, nil
}

func (p *ProductOperator) ProductByID(ctx context.Context, productID int64) (models.Product, error) {
	product, err := p.Redis.Product(ctx, productID)
	switch {
	case err == nil:
		return product, nil

	case !errors.Is(err, storage.ErrProductsNotFound):
		return models.Product{}, err
	}

	product, err = p.Postgres.ProductByID(ctx, productID)
	if err != nil {
		return models.Product{}, err
	}

	_ = p.Redis.SaveProduct(ctx, product)

	return product, nil
}

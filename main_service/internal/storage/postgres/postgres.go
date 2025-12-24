package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"main_service/internal/config"
	"main_service/internal/models"
	"main_service/internal/storage"

	"github.com/jackc/pgx"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepo struct {
	pool *pgxpool.Pool
}

func New(ctx context.Context, cfg *config.Config) (*PostgresRepo, error) {
	const op = "storage.postgres.New"

	dsn := dsn(cfg)

	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to parse config: %w", op, err)
	}

	poolConfig.MaxConns = 10
	poolConfig.MinConns = 2
	poolConfig.MaxConnLifetime = time.Hour
	poolConfig.MaxConnIdleTime = time.Minute * 30

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to create pool: %w", op, err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("%s: ping failed: %w", op, err)
	}

	return &PostgresRepo{pool: pool}, nil
}

// * SaveProduct добавляет продукт в базу данных
func (r *PostgresRepo) SaveProduct(ctx context.Context, userID int64, productURL, title string) (int64, error) {
	const op = "postgres.SaveProduct"

	const query = `
		INSERT INTO products (user_id, url, title)
		VALUES ($1, $2, $3)
		RETURNING id
	`

	var id int64

	err := r.pool.QueryRow(ctx, query, userID, productURL, title).Scan(&id)
	if err != nil {
		if pgErr, ok := err.(*pgx.PgError); ok && pgErr.Code == storage.UniqueViolation {
			return 0, storage.ErrUserAlreadyTracksProduct
		}

		return 0, fmt.Errorf("%s: failed to save product: %w", op, err)
	}

	return id, nil
}

// * Products возвращает слайс продуктов для вывода пользователю
func (r *PostgresRepo) Products(ctx context.Context, userID int64, limit, offset int) ([]models.Product, error) {
	const op = "postgres.Products"

	const query = `
		SELECT id, title, price, in_stock, last_checked, created_at, updated_at
		FROM products 
		WHERE user_id = $1 
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3;
	`

	rows, err := r.pool.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to get products: %w", op, err)
	}
	defer rows.Close()

	var products []models.Product
	for rows.Next() {
		var p models.Product

		err := rows.Scan(
			&p.ID,
			&p.Title,
			&p.Price,
			&p.In_stock,
			&p.Last_checked,
			&p.Created_at,
			&p.Updated_at,
		)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to scan products: %w", op, err)
		}

		products = append(products, p)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return products, nil
}

// * ProductByID возвращает продукт по ID
func (r *PostgresRepo) ProductByID(ctx context.Context, productID int64) (models.Product, error) {
	const op = "postgres.ProductByID"

	const query = `
		SELECT id, title, price, in_stock, last_checked, created_at, updated_at
		FROM products
		WHERE id = $1
	`

	row := r.pool.QueryRow(ctx, query, productID)

	var p models.Product

	err := row.Scan(
		&p.ID,
		&p.Title,
		&p.Price,
		&p.In_stock,
		&p.Last_checked,
		&p.Created_at,
		&p.Updated_at,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Product{}, storage.ErrProductsNotFound
		}

		return models.Product{}, fmt.Errorf("%s: failed to scan product: %w", op, err)
	}

	return p, nil
}

// * UpdateParsedData добавляет информацию о цене и наличии продукта
func (r *PostgresRepo) UpdateParsedData(ctx context.Context, productID int64, price int, inStock bool) error {
	const op = "postgres.UpdateParsedData"

	const query = `
		UPDATE products
		SET price = $1,
			in_stock = $2,
			last_checked = now(),
			updated_at = now()
		WHERE id = $3
	`

	cmd, err := r.pool.Exec(ctx, query, price, inStock, productID)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	if cmd.RowsAffected() == 0 {
		return storage.ErrProductsNotFound
	}

	return nil
}

// * DeleteProduct удаляет продукт по productID и userID
func (r *PostgresRepo) DeleteProduct(ctx context.Context, productID, userID int64) error {
	const op = "postgres.DeleteProduct"

	const query = `
		DELETE FROM products 
		WHERE id = $1 AND user_id = $2
	`

	cmd, err := r.pool.Exec(ctx, query, productID, userID)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	if cmd.RowsAffected() == 0 {
		return storage.ErrProductsNotFound
	}

	return nil
}

// * Close закрывает соединение с базой данных.
func (r *PostgresRepo) Close() {
	r.pool.Close()
}

// * dsn формирует конфигурацию базы данных.
func dsn(cfg *config.Config) string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s database=%s sslmode=%s",
		cfg.Postgres.Host,
		cfg.Postgres.Port,
		cfg.Postgres.User,
		cfg.Postgres.Password,
		cfg.Postgres.DBName,
		cfg.Postgres.SSLMode,
	)
}

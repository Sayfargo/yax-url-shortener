package url

import (
	"context"
	"database/sql"
	"log"
	"os"
	"testing"

	"github.com/Sayfargo/yax-url-shortener/migrations"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	pg "github.com/testcontainers/testcontainers-go/modules/postgres"
)

var testPool *pgxpool.Pool

func TestMain(m *testing.M) {
	ctx := context.Background()

	pgContainer, err := pg.Run(
		ctx,
		"postgres:18.6-alpine",
		pg.WithDatabase("db"),
		pg.WithUsername("postgres"),
		pg.WithPassword("postgres"),
		pg.BasicWaitStrategies(),
	)
	if err != nil {
		log.Fatalf("container run: %v", err)
	}

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		log.Fatalf("pg container connection string: %v", err)
	}

	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		log.Fatalf("pgxpool new: %v", err)
	}

	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("pool ping: %v", err)
	}

	db, err := sql.Open("pgx", connStr)
	if err != nil {
		log.Fatalf("sql open: %v", err)
	}

	goose.SetBaseFS(migrations.EmbedMigrations)

	if err := goose.SetDialect("postgres"); err != nil {
		log.Fatalf("goose set dialect: %v", err)
	}

	if err := goose.Up(db, "."); err != nil {
		pool.Close()
		log.Fatalf("goose up: %v", err)
	}

	testPool = pool

	code := m.Run()

	pool.Close()

	if err := db.Close(); err != nil {
		log.Fatalf("close db: %v", err)
	}

	if err := pgContainer.Terminate(ctx); err != nil {
		log.Fatalf("terminate container: %v", err)
	}

	os.Exit(code)
}

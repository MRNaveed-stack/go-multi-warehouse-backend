package config

import (
	"context"
	"database/sql"
	"log"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
)

var DB *sql.DB
var Redis *redis.Client

func ConnectDB() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading the .env file")

	}
	connStr := os.Getenv("DB_URL")

	db, err := sql.Open("pgx", connStr)
	if err != nil {
		log.Fatal(err)
	}
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)

	if err = db.Ping(); err != nil {
		log.Fatal(err)
	}
	DB = db
}

func ConnectRedis() {
	Redis = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})
	_, err := Redis.Ping(context.Background()).Result()
	if err != nil {
		log.Fatalf("Could not connect to Redis: %v", err)
	}
	log.Println("Successfully connected to Redis")
}

func Migrate(db *sql.DB) {
    queries := []string{
       `CREATE TABLE IF NOT EXISTS users (
            id SERIAL PRIMARY KEY,
            email TEXT UNIQUE NOT NULL,
            role TEXT NOT NULL,
            password TEXT NOT NULL,
            is_verified BOOLEAN DEFAULT FALSE,
            verification_token TEXT
        );`,

        `CREATE TABLE IF NOT EXISTS products (
            id SERIAL PRIMARY KEY,
            name TEXT NOT NULL,
            price DOUBLE PRECISION NOT NULL,
            quantity INTEGER NOT NULL,
           deleted_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
          min_stock_level INTEGER,
          version INTEGER,
         audit_logs TEXT

        );`,

        `CREATE TABLE IF NOT EXISTS stock_logs (
            id SERIAL PRIMARY KEY,
            product_name TEXT NOT NULL,
            change_amount INTEGER NOT NULL,
            user_email TEXT NOT NULL,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        );`,

        // ✅ audit_logs
        `CREATE TABLE IF NOT EXISTS audit_logs (
            id SERIAL PRIMARY KEY,
            user_id INTEGER REFERENCES users(id),
            action TEXT NOT NULL,
            entity_name TEXT NOT NULL,
            entity_id INTEGER,
            old_value JSONB,
            new_value JSONB,
            ip_address TEXT,
            created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
        );`,

        //  idempotency_keys
        `CREATE TABLE IF NOT EXISTS idempotency_keys (
            id_key TEXT PRIMARY KEY,
            response_code INTEGER,
            response_body TEXT,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            user_id INTEGER REFERENCES users(id)
        );`,

        //  warehouses
        `CREATE TABLE IF NOT EXISTS warehouses (
            id SERIAL PRIMARY KEY,
            name TEXT NOT NULL,
            location TEXT
        );`,

        //  stock_batches
        `CREATE TABLE IF NOT EXISTS stock_batches (
            id SERIAL PRIMARY KEY,
            product_id INTEGER REFERENCES products(id),
            warehouse_id INTEGER REFERENCES warehouses(id),
            batch_number TEXT NOT NULL,
            expiry_date TIMESTAMP,
            initial_quantity INTEGER NOT NULL,
            current_quantity INTEGER NOT NULL,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        );`,

        //  webhooks
        `CREATE TABLE IF NOT EXISTS webhooks (
            id SERIAL PRIMARY KEY,
            url TEXT NOT NULL,
            event_type TEXT NOT NULL,
            secret TEXT NOT NULL,
            is_active BOOLEAN DEFAULT TRUE,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        );`,
    }

    for _, q := range queries {
        _, err := db.Exec(q)
        if err != nil {
            log.Fatalf("Migration failed: %v", err)
        }
    }
    log.Println("Database migration completed successfully!")
}
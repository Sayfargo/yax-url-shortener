-- +goose Up
CREATE TABLE IF NOT EXISTS shortened_urls (
    uuid UUID PRIMARY KEY,
    short_code VARCHAR(20) UNIQUE NOT NULL,
    original_url TEXT
);

-- +goose Down
DROP TABLE IF EXISTS shorted_urls;
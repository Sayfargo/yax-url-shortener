-- +goose Up
CREATE UNIQUE INDEX IF NOT EXISTS idx_shortened_urls_original_url ON shortened_urls(original_url);

-- +goose Down
DROP INDEX IF EXISTS idx_shortened_urls_original_url;

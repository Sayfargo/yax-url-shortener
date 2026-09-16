-- +goose Up
ALTER TABLE shortened_urls 
ADD COLUMN is_deleted bool DEFAULT false;

CREATE INDEX idx_shortened_urls_lookup 
ON shortened_urls (short_code, user_id) 
WHERE is_deleted = false;

-- +goose Down
DROP INDEX IF EXISTS idx_shortened_urls_lookup;

ALTER TABLE shortened_urls
DROP COLUMN is_deleted;

-- +goose Up
ALTER TABLE shortened_urls 
ADD COLUMN user_id UUID;

CREATE INDEX idx_shortened_urls_user_id ON shortened_urls (user_id);

-- +goose Down
ALTER TABLE shortened_urls
DROP COLUMN user_id;

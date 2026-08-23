include .env
export

SERVER_DIR=cmd/shortener
BINARY_NAME=shortener.exe
TESTER_NAME=shortenertest.exe


num=1

.PHONY: build test clean

build: 
	go build -o $(SERVER_DIR)/$(BINARY_NAME) $(SERVER_DIR)/main.go

test:
	$(MAKE) build
	./$(TESTER_NAME) \
		-test.v \
		-test.run=^TestIteration$(num)$$ \
		-binary-path=$(SERVER_DIR)/$(BINARY_NAME) \
		-source-path=.

clean:
	rm -f $(SERVER_DIR)/$(BINARY_NAME)
	@echo "Cleared!"

goose-create:
	@if [ -z "$(name)" ]; then \
		echo "There is no name param. Example: make migrate-create name=name_of_migrate"; \
		exit 1; \
	fi;	\

	goose \
	create $(name) sql

goose-status:
	goose \
	status


goose-up:
	goose \
	up

goose-up-by-one:
	goose \
	up-by-one

goose-down:
	goose \
	down

goose-down-to: 
	@if [ -z "$(id)" ]; then \
		echo "There is no migrate id. Example: make goose-down-to id=20170614145246"; \
		exit 1; \
	fi; \

	goose \
	down-to $(id)

goose-up-to:
	@if [ -z "$(id)" ]; then \
		echo "There is no migrate id. Example: make goose-down-to id=20170614145246"; \
		exit 1; \
	fi;	\

	goose \
	up-to $(id)
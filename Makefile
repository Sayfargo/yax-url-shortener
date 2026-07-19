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
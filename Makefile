GO_BUILD_ENV := CGO_ENABLED=0 GOOS=linux GOARCH=amd64
BIN_DIR := bin
BIN_PATH := $(BIN_DIR)/paas-backend

build:
	mkdir -p $(BIN_DIR)
	$(GO_BUILD_ENV) go build -v -o $(BIN_PATH) cmd/my-go-backend/main.go

clean:
	rm -rf $(BIN_DIR)

heroku: build
	heroku container:push web
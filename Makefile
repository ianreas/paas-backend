# GO_BUILD_ENV := CGO_ENABLED=0 GOOS=linux GOARCH=amd64
# BIN_DIR := bin
# BIN_PATH := $(BIN_DIR)/paas-backend

# build:
# 	mkdir -p $(BIN_DIR)
# 	$(GO_BUILD_ENV) go build -v -o $(BIN_PATH) cmd/my-go-backend/main.go

# clean:
# 	rm -rf $(BIN_DIR)

#  build:
#    	mkdir -p bin
#    	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
#    	go build -v -o bin/paas-backend cmd/my-go-backend/main.go
# Makefile

GO_BUILD_ENV := CGO_ENABLED=0 GOOS=linux GOARCH=amd64
BIN_DIR := bin
BIN_PATH := $(BIN_DIR)/paas-backend

build:
	mkdir -p $(BIN_DIR)
	$(GO_BUILD_ENV) go build -v -o $(BIN_PATH) cmd/my-go-backend/main.go

clean:
	rm -rf $(BIN_DIR)

docker-build:
	docker build -t my-paas-backend:latest .

docker-push:
	docker push my-paas-backend:latest
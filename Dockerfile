FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/paas-backend cmd/my-go-backend/main.go

FROM alpine:latest
RUN apk add --no-cache docker-cli
WORKDIR /app
COPY --from=builder /app/bin/paas-backend .
EXPOSE 3005
CMD ["./paas-backend"]
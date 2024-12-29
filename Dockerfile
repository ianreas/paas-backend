   # Stage 1: build
   FROM golang:1.20 AS builder
   WORKDIR /src
   COPY go.mod go.sum ./
   RUN go mod download
   COPY . ./
   RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
       go build -o bin/paas-backend cmd/my-go-backend/main.go

   # Stage 2: final, minimal image
   FROM alpine:latest
   WORKDIR /app
   COPY --from=builder /src/bin/paas-backend .
   EXPOSE 3000
   CMD ["./paas-backend"]
FROM golang:alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o goretro main.go

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/goretro .
COPY --from=builder /app/static ./static

EXPOSE 8080

CMD ["./goretro"]

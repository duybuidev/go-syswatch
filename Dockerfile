# Dùng bản Go mới nhất hiện có trên Docker Hub
FROM golang:alpine AS builder

WORKDIR /app

# Bật tự động toolchain
ENV GOTOOLCHAIN=auto

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Biên dịch tĩnh
RUN CGO_ENABLED=0 GOOS=linux go build -o syswatch cmd/syswatch/main.go

# Đóng gói siêu nhẹ
FROM alpine:latest
WORKDIR /root/
RUN apk --no-cache add ca-certificates tzdata
COPY --from=builder /app/syswatch .
CMD ["./syswatch"]

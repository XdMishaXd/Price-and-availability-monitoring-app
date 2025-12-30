# * Build Stage
FROM golang:latest-alpine AS builder

LABEL image_author="Michael Prunchak"

ARG VERSION=dev
ARG BUILD_TIME
ARG COMMIT_SHA

RUN apk add --no-cache \
	git \
	ca-certificates \
	tzdata

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .

# Сборка бинарника
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
	-ldflags="-w -s \
	-X main.Version=${VERSION} \
	-X main.BuildTime=${BUILD_TIME} \
	-X main.CommitSHA=${COMMIT_SHA}" \
	-trimpath \
	-o /build/app \
	./cmd/main_service/main.go

# Проверка размера бинарника
RUN size /build/app

# * Runtime Stage
FROM alpine:latest

# Установка зависимостей
RUN apk add --no-cache \
	ca-certificates \
	tzdata \
	curl \
	&& adduser -D -g '' appuser

WORKDIR /app

COPY --from=builder /build/app /app/main_service

COPY --from=builder /build/config /app/config

RUN mkdir -p /app/logs /app/data && \
	chown -R appuser:appuser /app

USER appuser

EXPOSE 8080

# Healthcheck
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
	CMD curl -f http://localhost:8080/health || exit 1

# Метаданные
LABEL version="${VERSION}"
LABEL build.time="${BUILD_TIME}"
LABEL git.commit="${COMMIT_SHA}"

# Запуск
ENTRYPOINT ["/app/main_service"]

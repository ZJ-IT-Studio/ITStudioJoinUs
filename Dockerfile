FROM node:24-alpine AS web
WORKDIR /src/web
COPY web/package*.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

FROM golang:1.26-alpine AS api
WORKDIR /src
RUN apk add --no-cache ca-certificates tzdata
COPY go.mod go.sum ./
RUN go mod download
COPY server/ ./server/
COPY --from=web /src/web/dist/ ./server/webui/dist/
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/itstudio ./server/cmd/server

FROM alpine:3.22
RUN apk add --no-cache ca-certificates tzdata && addgroup -S app && adduser -S app -G app \
    && mkdir -p /data/uploads /data/backups && chown -R app:app /data
USER app
WORKDIR /app
COPY --from=api /out/itstudio /app/itstudio
ENV APP_ADDR=:8080 DATA_DIR=/data COOKIE_SECURE=true
EXPOSE 8080
VOLUME ["/data"]
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s CMD wget -q -O - http://127.0.0.1:8080/healthz || exit 1
ENTRYPOINT ["/app/itstudio"]


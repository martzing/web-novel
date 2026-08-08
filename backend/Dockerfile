FROM golang:1.25-alpine AS build
WORKDIR /src

RUN apk add --no-cache git

COPY go.mod go.sum* ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/api      ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/migrate  ./cmd/migrate
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/worker   ./cmd/worker

FROM alpine:3.20
# tzdata is required: the nightly bonus expiry is pinned to Asia/Bangkok.
RUN apk add --no-cache tzdata && adduser -D -u 10001 app
WORKDIR /app
COPY --from=build /out/api     /app/api
COPY --from=build /out/migrate /app/migrate
COPY --from=build /out/worker  /app/worker
RUN mkdir -p /app/uploads && chown -R app:app /app
USER app
EXPOSE 8080
CMD ["/app/api"]

FROM golang:1.24.0-alpine AS builder

RUN addgroup -S nonroot \
    && adduser -S nonroot -G nonroot

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY src ./src

# Disables CGO and specifies the name for the compiled application as app
RUN CGO_ENABLED=0 GOOS=linux go build -o app ./src/backend

FROM alpine:3.21.3

RUN addgroup -S nonroot \
    && adduser -S nonroot -G nonroot

USER nonroot

COPY --from=builder /app/app /app/app

WORKDIR /app

COPY src /app/src

EXPOSE 8080

ENTRYPOINT ["/app/app"]
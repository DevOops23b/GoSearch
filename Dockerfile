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

WORKDIR /app

COPY --from=builder /app/app /app/app
COPY src /app/src

RUN mkdir -p /app/frontend/templates

RUN find /app -type d | sort

RUN if [ -d "/app/src/frontend/templates" ]; then \
        cp -r /app/src/frontend/templates/* /app/frontend/templates/; \
    else \
        echo "Templates directory not found at expected location"; \
        find /app -name "*.html" | sort; \
    fi

USER nonroot

EXPOSE 8080

ENTRYPOINT ["/app/app"]
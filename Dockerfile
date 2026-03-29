# syntax=docker/dockerfile:1

FROM node:20-alpine AS web-builder
WORKDIR /app
COPY . .
RUN if [ -f web/package.json ]; then \
      cd web && npm ci && npm run build; \
    else \
      mkdir -p web/dist; \
    fi

FROM golang:1.26-alpine AS go-builder
WORKDIR /src
COPY --from=web-builder /app/web/dist ./web/dist
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/apicerberus ./cmd/apicerberus

FROM alpine:3.20
WORKDIR /app
RUN adduser -D -u 10001 apicerberus
COPY --from=go-builder /out/apicerberus /usr/local/bin/apicerberus
USER apicerberus
EXPOSE 8080 8443
ENTRYPOINT ["/usr/local/bin/apicerberus"]

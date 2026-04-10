# Build web assets
FROM node:22-alpine AS ui
WORKDIR /src/webapp
COPY webapp/package*.json ./
RUN npm ci
COPY webapp/ ./
RUN npm run build

# Build Go binary
FROM golang:1.25-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=ui /src/ui/dist ./ui/dist
RUN apk add --no-cache build-base \
 && CGO_ENABLED=1 go build -o lanmapper ./cmd/lanmapper

# Runtime image
FROM alpine:3.19
RUN addgroup -S lanmapper && adduser -S lanmapper -G lanmapper
WORKDIR /app
COPY --from=builder /src/lanmapper ./lanmapper
COPY scripts/entrypoint.sh ./entrypoint.sh
RUN mkdir -p /app/data && chown -R lanmapper:lanmapper /app && chmod +x entrypoint.sh
USER lanmapper
EXPOSE 8080
ENTRYPOINT ["/app/entrypoint.sh"]

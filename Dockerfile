FROM golang:1.26-alpine AS builder

RUN apk add --no-cache gcc musl-dev

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=dev
RUN CGO_ENABLED=1 go build -ldflags "-s -w -X main.version=${VERSION}" -o /strix .

FROM alpine:latest

RUN apk add --no-cache ffmpeg ca-certificates

COPY --from=builder /strix /usr/local/bin/strix

WORKDIR /app
COPY cameras.db .

EXPOSE 4567

HEALTHCHECK --interval=30s --timeout=3s CMD wget -q --spider http://localhost:4567/api/health || exit 1

USER nobody
ENTRYPOINT ["strix"]

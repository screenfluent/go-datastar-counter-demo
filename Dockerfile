# syntax=docker/dockerfile:1

FROM golang:1.26.2-alpine AS build

RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go install github.com/a-h/templ/cmd/templ@v0.3.1001
RUN templ generate
RUN go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

FROM alpine:3.22

RUN apk add --no-cache ca-certificates
WORKDIR /app

COPY --from=build /out/server /app/server
COPY migrations /app/migrations
COPY static /app/static

ENV PORT=8080
EXPOSE 8080

CMD ["/app/server"]

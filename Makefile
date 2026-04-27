.PHONY: generate sqlc dev build docker run test tidy

generate:
	go tool templ generate

sqlc:
	go tool sqlc generate

dev: generate
	go run ./cmd/server

build: generate
	go build ./cmd/server

docker:
	docker build -t go-datastar-counter-demo .

run:
	docker run --rm -p 8080:8080 go-datastar-counter-demo

test:
	go test ./...

tidy:
	go mod tidy


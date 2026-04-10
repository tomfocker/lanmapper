BINARY=lanmapper
.PHONY: build run ui dev
build:
	GOOS=linux GOARCH=amd64 go build -o bin/$(BINARY) ./cmd/lanmapper
run:
	go run ./cmd/lanmapper
ui:
	cd webapp && npm install && npm run dev

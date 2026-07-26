.PHONY: build run test ui-install ui-build ui-dev

build: ui-build
	go build -o bin/microscope ./cmd/server

run: build
	./bin/microscope

test:
	go test ./...

ui-install:
	cd ui && npm ci

ui-build:
	cd ui && npm run build

ui-dev:
	cd ui && npm run dev

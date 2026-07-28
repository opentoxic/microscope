.PHONY: sync-core build run test ui-install ui-build ui-dev test-all

sync-core:
	sh scripts/sync-core-assets.sh

ui-install:
	cd core/ui && pnpm install --frozen-lockfile

ui-build: ui-install
	cd core/ui && pnpm run build
	$(MAKE) sync-core

ui-dev:
	cd core/ui && pnpm run dev

build: sync-core
	cd adaptor/go && go build -o ../../bin/microscope ./cmd/server

run: build
	./bin/microscope

test:
	cd adaptor/go && go test ./...

test-php:
	cd adaptor/php && composer test

test-python:
	cd adaptor/python && python -m pytest

test-all: test test-php test-python

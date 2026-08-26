APP := boiler-energy-efficiency-service
BINARY := bin/$(APP)
GO ?= go
PORT ?= 8080

.PHONY: build test vet fmt run clean docker-build docker-run

build:
	$(GO) build -o $(BINARY) .

test:
	$(GO) test ./...

test-race:
	$(GO) test -race ./...

vet:
	$(GO) vet ./...

fmt:
	gofmt -w .

run:
	PORT=$(PORT) $(GO) run .

clean:
	rm -rf bin data

docker-build:
	docker build -t $(APP):latest .

docker-run:
	docker run --rm -p $(PORT):8080 -e PORT=8080 $(APP):latest

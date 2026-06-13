BIN     := out/cakrawala
CONFIG  ?= config.yaml
ARGS    ?=

.PHONY: build run dry-run start test tidy clean

build:
	mkdir -p out
	go build -o $(BIN) ./cmd/...

run: build
	./$(BIN) run -c $(CONFIG) $(ARGS)

dry-run: build
	./$(BIN) run --dry-run -c $(CONFIG) $(ARGS)

start: build
	./$(BIN) start -c $(CONFIG)

test:
	go test ./...

tidy:
	go mod tidy

clean:
	rm -rf out

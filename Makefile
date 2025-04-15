build:
	@go build -o="./tmp/tui" ./cmd/tui/

run: build
	@./tmp/tui



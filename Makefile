.PHONY: wasm-bin
wasm-bin:
	GOOS=js GOARCH=wasm go build -o ./dist/demo.wasm ./cmd/wasm

.PHONY: server
server:
	go run .

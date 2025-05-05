install:
	go mod tidy

build:
	go build -o ndns-go ./pkg/main.go

run:
	go run ./pkg/main.go




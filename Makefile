

cmd: cmd/main.go
	go build -o foobarhttp cmd/main.go

.PHONY: run

run:
	go run cmd/main.go
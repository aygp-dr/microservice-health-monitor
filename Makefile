.PHONY: build run test clean

build:
	go build -o bin/microservice-health-monitor .

run: build
	./bin/microservice-health-monitor

test:
	go test ./...

clean:
	rm -rf bin/

all: build

build:
	go build .

tidy:
	go mod tidy

clean:
	rm sources-superkey-worker

container:
	docker build . -t sources-superkey-worker -f Dockerfile

run: build
	./sources-superkey-worker

inlinerun:
	go run .

fancyrun: build
	./sources-superkey-worker | grep '^{' | jq -r .

runcontainer: container
	docker run -ti --rm --net host -e KAFKA_BROKERS=localhost:9092 sources-superkey-worker

remotedebug:
	dlv debug --headless --listen=:2345 --api-version=2 --accept-multiclient

debug:
	dlv debug

lint:
	go vet ./...
	golangci-lint run -E gofmt,gci,bodyclose,forcetypeassert,misspell

gci:
	golangci-lint run -E gci --fix

.PHONY: build container run fancyrun runcontainer clean debug tidy debug remotedebug lint gci

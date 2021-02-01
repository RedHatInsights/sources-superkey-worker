all: build

build:
	go build .

container:
	docker build . -t sources-superkey-worker -f Containerfile

run: build
	./sources-superkey-worker

runcontainer: container
	docker run -ti --rm --net host -e KAFKA_BROKERS=localhost:9092 sources-superkey-worker

.PHONY: build container

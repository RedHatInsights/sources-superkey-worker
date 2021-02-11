all: build

build:
	go build .

clean:
	rm sources-superkey-worker

container:
	docker build . -t sources-superkey-worker -f Containerfile

run: build
	./sources-superkey-worker

fancyrun: build
	./sources-superkey-worker | grep '^{' | jq -r .

runcontainer: container
	docker run -ti --rm --net host -e KAFKA_BROKERS=localhost:9092 sources-superkey-worker

.PHONY: build container run fancyrun runcontainer clean

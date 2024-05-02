FROM registry.access.redhat.com/ubi9/ubi-minimal:latest as build
WORKDIR /build

RUN microdnf -y install go

COPY go.mod .
RUN go mod download

COPY . .
RUN go build

FROM registry.access.redhat.com/ubi9/ubi-minimal:latest
COPY --from=build /build/sources-superkey-worker /sources-superkey-worker

ENTRYPOINT ["/sources-superkey-worker"]

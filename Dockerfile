FROM registry.access.redhat.com/ubi8/ubi-minimal:latest as build
WORKDIR /build

RUN microdnf install go

COPY go.mod .
RUN go mod download

COPY . .
RUN go build

FROM registry.access.redhat.com/ubi8/ubi-minimal:latest
COPY --from=build /build/sources-superkey-worker /sources-superkey-worker

ENTRYPOINT ["/sources-superkey-worker"]

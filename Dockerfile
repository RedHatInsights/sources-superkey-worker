FROM registry.access.redhat.com/ubi8/ubi-minimal:8.5-218 as build
WORKDIR /build

RUN microdnf install go

COPY go.mod .
RUN go mod download 

COPY . .
RUN go build

FROM registry.access.redhat.com/ubi8/ubi-minimal:8.5-218
COPY --from=build /build/sources-superkey-worker /sources-superkey-worker

ENTRYPOINT ["/sources-superkey-worker"]

FROM registry.access.redhat.com/ubi9/ubi-minimal:latest as build
WORKDIR /build

RUN microdnf install go

# We need to override the toolchain to the latest version because
# unfortunately the latest "ubi8" image does not contain the go version 1.23,
# which is required for the latest dependency updates.

COPY go.mod .
RUN GOTOOLCHAIN=go1.23.5 go mod download

COPY . .
RUN GOTOOLCHAIN=go1.23.5 go build

FROM registry.access.redhat.com/ubi9/ubi-minimal:latest
COPY --from=build /build/sources-superkey-worker /sources-superkey-worker

COPY licenses/LICENSE /licenses/LICENSE

USER 1001

ENTRYPOINT ["/sources-superkey-worker"]

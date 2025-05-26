FROM registry.access.redhat.com/ubi9/ubi-minimal:latest as build
WORKDIR /build

RUN microdnf install --assumeyes go \
    && microdnf clean all

# We need to override the toolchain to the latest version because
# unfortunately the latest "ubi8" image does not contain the go version 1.23,
# which is required for the latest dependency updates.
ARG GOTOOLCHAIN=go1.24.3

COPY . .
RUN go mod download \
    && go build

FROM registry.access.redhat.com/ubi9/ubi-minimal:latest
COPY --from=build /build/sources-superkey-worker /sources-superkey-worker

COPY licenses/LICENSE /licenses/LICENSE

USER 1001

ENTRYPOINT ["/sources-superkey-worker"]

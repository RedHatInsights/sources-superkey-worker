FROM registry.access.redhat.com/ubi9/ubi-minimal:latest as build
WORKDIR /build

RUN microdnf install --assumeyes go \
    && microdnf clean all

COPY . .
RUN go mod download \
    && go build

FROM registry.access.redhat.com/ubi9/ubi-minimal:latest
COPY --from=build /build/sources-superkey-worker /sources-superkey-worker

COPY licenses/LICENSE /licenses/LICENSE

USER 1001

ENTRYPOINT ["/sources-superkey-worker"]

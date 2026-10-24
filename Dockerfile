FROM registry.access.redhat.com/ubi9/ubi-minimal:latest@sha256:2d4c4551af6831dcd26bbf86108c4385de5183ec8fe6b5868f6f9640fe47f4ed as build
WORKDIR /build

RUN microdnf install --assumeyes go \
    && microdnf clean all

COPY . .
RUN go mod download \
    && go build

FROM registry.access.redhat.com/ubi9/ubi-minimal:latest@sha256:2d4c4551af6831dcd26bbf86108c4385de5183ec8fe6b5868f6f9640fe47f4ed
COPY --from=build /build/sources-superkey-worker /sources-superkey-worker

COPY licenses/LICENSE /licenses/LICENSE

USER 1001

ENTRYPOINT ["/sources-superkey-worker"]

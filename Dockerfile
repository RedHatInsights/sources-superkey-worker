FROM registry.access.redhat.com/ubi9/ubi-minimal:latest@sha256:6f3a78179e0b30b930a186e7f68a4af52e51bb82406c9653be10bddd624e40f5 as build
WORKDIR /build

RUN microdnf install --assumeyes go \
    && microdnf clean all

COPY . .
RUN go mod download \
    && go build

FROM registry.access.redhat.com/ubi9/ubi-minimal:latest@sha256:6f3a78179e0b30b930a186e7f68a4af52e51bb82406c9653be10bddd624e40f5
COPY --from=build /build/sources-superkey-worker /sources-superkey-worker

COPY licenses/LICENSE /licenses/LICENSE

USER 1001

ENTRYPOINT ["/sources-superkey-worker"]

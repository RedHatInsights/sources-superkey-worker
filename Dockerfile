################################
# STEP 1 build executable binary
################################
FROM registry.access.redhat.com/hi/go:latest-fips-builder@sha256:9b159a6371ac623f7a9170459efd2a3d96ab34ca37c3fcf215043865e1a40d0c AS builder

USER 0

WORKDIR /workspace

# Cache deps before copying source so that we do not need to re-download for every build
COPY go.mod go.sum .

# Fetch dependencies hermetically by sourcing the local prefetch cache
RUN if [ -f /cachi2/cachi2.env ]; then . /cachi2/cachi2.env; fi && \
    go mod download

# Now copy the rest of the files for build
COPY *.go .
COPY amazon amazon
COPY config config
COPY logger logger
COPY provider provider
COPY sources sources
COPY superkey superkey

# Build the binary using the local cached modules
RUN if [ -f /cachi2/cachi2.env ]; then . /cachi2/cachi2.env; fi && \
    go build -ldflags "-w -s" -o sources-superkey-worker .

############################
# STEP 2 build a small image
############################
FROM registry.access.redhat.com/hi/go:latest-fips@sha256:071fc6c7b99f743b61f10f837bb72c9c1c1528193e0665e923b194c4a75535af

WORKDIR /

COPY --from=builder /workspace/sources-superkey-worker /sources-superkey-worker

COPY licenses/LICENSE /licenses/LICENSE

USER 1001

ENTRYPOINT ["/sources-superkey-worker"]

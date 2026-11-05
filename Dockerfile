################################
# STEP 1 build executable binary
################################
FROM registry.access.redhat.com/hi/go:latest-fips-builder@sha256:9c8fb40ab55e721ddc35370e3c5d37969f5a65290973263e7874eecb62482834 AS builder

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
FROM registry.access.redhat.com/hi/go:latest-fips@sha256:04e15ddcf64b1d72d618d7ab3f02a8ff90bd1607ba500b7e31fcaa007bbcec07

WORKDIR /

COPY --from=builder /workspace/sources-superkey-worker /sources-superkey-worker

COPY licenses/LICENSE /licenses/LICENSE

USER 1001

ENTRYPOINT ["/sources-superkey-worker"]

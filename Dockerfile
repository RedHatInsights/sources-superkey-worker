FROM registry.access.redhat.com/ubi8/ubi-minimal:8.3-291 as build
MAINTAINER jlindgre@redhat.com

RUN mkdir /build
WORKDIR /build

RUN microdnf install go

COPY go.mod .
RUN go mod download 

COPY . .
RUN go build

FROM registry.access.redhat.com/ubi8/ubi-minimal:8.3-291
COPY --from=build /build/sources-superkey-worker /sources-superkey-worker

RUN curl -L -o /usr/bin/haberdasher \
    https://github.com/RedHatInsights/haberdasher/releases/latest/download/haberdasher_linux_amd64 && \
    chmod 755 /usr/bin/haberdasher

ENTRYPOINT ["/usr/bin/haberdasher"]
CMD ["/sources-superkey-worker"]

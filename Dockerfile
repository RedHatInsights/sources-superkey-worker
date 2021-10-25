FROM registry.access.redhat.com/ubi8/ubi-minimal:8.4-210 as build

RUN mkdir /build
WORKDIR /build

RUN microdnf install go

COPY go.mod .
RUN go mod download

COPY . .
RUN go build

FROM registry.access.redhat.com/ubi8/ubi-minimal:8.4-210
COPY --from=build /build/sources-superkey-worker /sources-superkey-worker

# install az cli from micro$oft
RUN rpm --import https://packages.microsoft.com/keys/microsoft.asc && \
    echo -e "[azure-cli]\nname=Azure CLI\nbaseurl=https://packages.microsoft.com/yumrepos/azure-cli\nenabled=1\ngpgcheck=1\ngpgkey=https://packages.microsoft.com/keys/microsoft.asc" | tee /etc/yum.repos.d/azure-cli.repo && \
    microdnf install azure-cli

ENTRYPOINT ["/sources-superkey-worker"]

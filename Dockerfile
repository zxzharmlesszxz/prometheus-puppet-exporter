FROM golang:1.27.1-alpine3.24 AS build

WORKDIR /src

ARG TARGETOS
ARG TARGETARCH
ARG VERSION
ARG BRANCH
ARG REVISION
ARG BUILD_USER
ARG BUILD_DATE
ARG BUILDER_PACKAGES="make"

RUN apk add --no-cache --upgrade ${BUILDER_PACKAGES}

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN make build \
    BUILD_OUTPUT=dist/prometheus-puppet-exporter \
    GOOS=${TARGETOS} \
    GOARCH=${TARGETARCH} \
    ${VERSION:+VERSION="${VERSION}"} \
    ${BRANCH:+BRANCH="${BRANCH}"} \
    ${REVISION:+REVISION="${REVISION}"} \
    ${BUILD_USER:+BUILD_USER="${BUILD_USER}"} \
    ${BUILD_DATE:+BUILD_DATE="${BUILD_DATE}"}

FROM alpine:3.24 AS base

ARG RUNTIME_PACKAGES="ca-certificates libcrypto3 libssl3"
RUN apk add --no-cache --upgrade ${RUNTIME_PACKAGES}

FROM base

COPY --from=build /src/dist/prometheus-puppet-exporter /usr/local/bin/prometheus-puppet-exporter

EXPOSE 9819

USER nobody

HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD wget -q -T 2 --spider http://127.0.0.1:9819/healthz || exit 1

ENTRYPOINT ["/usr/local/bin/prometheus-puppet-exporter"]

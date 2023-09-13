FROM --platform=${BUILDPLATFORM:-linux/amd64} golang:1.17 as builder

ARG TARGETPLATFORM
ARG BUILDPLATFORM
ARG TARGETOS
ARG TARGETARCH

WORKDIR /app
COPY . /app
RUN \
    set -ex \
    && GO111MODULE=on go get -v ./... \
    && CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -ldflags="-w -s" \
        -o envars-webhook ./cmd/envars-webhook

FROM --platform=${TARGETPLATFORM:-linux/amd64} scratch
WORKDIR /
COPY --from=builder /app/envars-webhook /envars-webhook
ENTRYPOINT ["/envars-webhook"]

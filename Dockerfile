FROM ubuntu:latest

ARG TARGETOS
ARG TARGETARCH
COPY target/envars-webhook_${TARGETOS}_${TARGETARCH} /envars-webhook
ENTRYPOINT ["tail", "-f", "/dev/null"]


FROM scratch

ARG TARGETOS
ARG TARGETARCH
COPY target/envars-webhook_${TARGETOS}_${TARGETARCH} /envars-webhook
ENTRYPOINT ["/envars-webhook"]

FROM scratch

COPY ./envars-webhook /
ENTRYPOINT ["/envars-webhook"]

# change
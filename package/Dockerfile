FROM ubuntu:26.04@sha256:2260313b31c8c011cd2eebe728008efac1b3982be73eb71348ea2648d2c0e09b

ARG PASTURESTACK_IMAGE_VERSION=dev

LABEL org.opencontainers.image.source="https://github.com/PastureStack/node-agent" \
      org.opencontainers.image.description="PastureStack node agent compatibility image." \
      org.opencontainers.image.licenses="Apache-2.0" \
      org.opencontainers.image.version="${PASTURESTACK_IMAGE_VERSION}"

RUN rm -f /usr/bin/pebble
COPY node-agent /usr/bin/
CMD ["node-agent"]

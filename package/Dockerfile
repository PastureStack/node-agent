FROM ubuntu:26.04

ARG PASTURESTACK_IMAGE_VERSION=dev

LABEL org.opencontainers.image.source="https://github.com/PastureStack/node-agent" \
      org.opencontainers.image.description="PastureStack node agent compatibility image." \
      org.opencontainers.image.licenses="Apache-2.0" \
      org.opencontainers.image.version="${PASTURESTACK_IMAGE_VERSION}"

RUN rm -f /usr/bin/pebble
COPY node-agent /usr/bin/
CMD ["node-agent"]

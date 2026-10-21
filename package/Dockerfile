FROM ubuntu:26.04@sha256:2260313b31c8c011cd2eebe728008efac1b3982be73eb71348ea2648d2c0e09b

ARG PASTURESTACK_IMAGE_VERSION=dev

LABEL org.opencontainers.image.source="https://github.com/PastureStack/node-agent" \
      org.opencontainers.image.description="PastureStack node agent compatibility image." \
      org.opencontainers.image.licenses="Apache-2.0" \
      org.opencontainers.image.version="${PASTURESTACK_IMAGE_VERSION}"

RUN rm -f /usr/bin/pebble
COPY libssl3t64.deb openssl-provider-legacy.deb /tmp/
RUN set -eu; \
    for package in libssl3t64 openssl-provider-legacy; do \
        test "$(dpkg-deb -f "/tmp/${package}.deb" Package)" = "$package"; \
        test "$(dpkg-deb -f "/tmp/${package}.deb" Version)" = 3.5.5-1ubuntu3.7; \
        test "$(dpkg-deb -f "/tmp/${package}.deb" Architecture)" = amd64; \
    done; \
    dpkg -i /tmp/libssl3t64.deb /tmp/openssl-provider-legacy.deb; \
    for package in libssl3t64 openssl-provider-legacy; do \
        test "$(dpkg-query -W -f='${Version}' "$package")" = 3.5.5-1ubuntu3.7; \
        rm -f "/tmp/${package}.deb"; \
    done
COPY node-agent /usr/bin/
CMD ["node-agent"]

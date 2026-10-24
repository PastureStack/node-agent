# syntax=docker/dockerfile:1.7
FROM ghcr.io/pasturestack/ubuntu-security-backports:v0.1.0@sha256:2742a3d429e4e14bc587f6dedc51ded4246cf90bbaedcd86c67d0885e32a6f2a AS ubuntu_security_backports
FROM ubuntu:26.04@sha256:2260313b31c8c011cd2eebe728008efac1b3982be73eb71348ea2648d2c0e09b AS certificates
COPY ca-certificates.deb /tmp/ca-certificates.deb
RUN set -eu; \
    test "$(dpkg-deb -f /tmp/ca-certificates.deb Version)" = 20260816ubuntu0.26.04.1; \
    dpkg-deb --extract /tmp/ca-certificates.deb /ca; \
    mkdir -p /out; \
    find /ca/usr/share/ca-certificates -type f -name '*.crt' | LC_ALL=C sort | \
        while IFS= read -r certificate; do sed -e '$a\' "$certificate"; done > /out/ca-certificates.crt; \
    test -s /out/ca-certificates.crt; \
    test ! -e /ca/usr/share/ca-certificates/mozilla/emSign_ECC_Root_CA_-_G3.crt

FROM ubuntu:26.04@sha256:2260313b31c8c011cd2eebe728008efac1b3982be73eb71348ea2648d2c0e09b

COPY --from=certificates /out/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=certificates /ca/usr/share/doc/ca-certificates/ /licenses/ca-certificates/
COPY ubuntu-security-packages.lock /licenses/ubuntu-security-packages.lock
COPY install-ubuntu-security-updates /usr/local/libexec/pasturestack/install-ubuntu-security-updates

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
RUN sh /usr/local/libexec/pasturestack/install-ubuntu-security-updates \
        /licenses/ubuntu-security-packages.lock /etc/ssl/certs/ca-certificates.crt
COPY node-agent /usr/bin/
# Install qualified backports only for packages already in this runtime.
# The immutable read-only build mount adds no installer or interpreter to it.
RUN --mount=from=ubuntu_security_backports,source=/out,target=/opt/pasturestack/ubuntu-security-backports,ro \
    set -eux; \
    evidence=/usr/share/pasturestack/security/ubuntu-backports; \
    mkdir -p "${evidence}"; \
    /opt/pasturestack/ubuntu-security-backports/install-existing-backports --install-in-docker-build > "${evidence}/install.json"; \
    /opt/pasturestack/ubuntu-security-backports/install-existing-backports --verify-installed > "${evidence}/verify.json"; \
    for document in manifest.json qualification.json openvex.json source-inputs.lock.json; do \
        cp "/opt/pasturestack/ubuntu-security-backports/${document}" "${evidence}/${document}"; \
    done; \
    printf '%s\n' 'ghcr.io/pasturestack/ubuntu-security-backports:v0.1.0@sha256:2742a3d429e4e14bc587f6dedc51ded4246cf90bbaedcd86c67d0885e32a6f2a' > "${evidence}/bundle-image.txt"; \
    (cd "${evidence}" && sha256sum manifest.json qualification.json openvex.json source-inputs.lock.json install.json verify.json bundle-image.txt > SHA256SUMS); \
    dpkg-query -W -f='${binary:Package}\t${Version}\n' | LC_ALL=C sort > /usr/share/pasturestack/ubuntu-security-packages.tsv; \
    rm -f /var/cache/ldconfig/aux-cache

CMD ["node-agent"]

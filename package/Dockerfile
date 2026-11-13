# syntax=docker/dockerfile:1.7
FROM ghcr.io/pasturestack/ubuntu-security-backports:v0.1.0@sha256:2742a3d429e4e14bc587f6dedc51ded4246cf90bbaedcd86c67d0885e32a6f2a AS ubuntu_security_backports
FROM ubuntu:26.04@sha256:2260313b31c8c011cd2eebe728008efac1b3982be73eb71348ea2648d2c0e09b AS certificates
COPY ca-certificates.deb /tmp/ca-certificates.deb
RUN set -eu; \
    test "$(dpkg-deb -f /tmp/ca-certificates.deb Version)" = 20260816ubuntu0.26.04.2; \
    dpkg-deb --extract /tmp/ca-certificates.deb /ca; \
    mkdir -p /out; \
    find /ca/usr/share/ca-certificates -type f -name '*.crt' | LC_ALL=C sort | \
        while IFS= read -r certificate; do sed -e '$a\' "$certificate"; done > /out/ca-certificates.crt; \
    test -s /out/ca-certificates.crt; \
    test ! -e /ca/usr/share/ca-certificates/mozilla/emSign_ECC_Root_CA_-_G3.crt

FROM ubuntu:26.04@sha256:2260313b31c8c011cd2eebe728008efac1b3982be73eb71348ea2648d2c0e09b AS qualified_backports

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
# Official fixed APT pair, before the runtime's existing micro-update calls.
COPY apt.deb libapt-pkg7.0.deb /tmp/
RUN set -eu; \
    . /etc/os-release; test "$ID:$VERSION_ID" = ubuntu:26.04; \
    before=$(mktemp); after=$(mktemp); trap 'rm -f "$before" "$after"' EXIT; \
    dpkg-query -W -f='${binary:Package}\t${db:Status-Status}\n' | LC_ALL=C sort > "$before"; \
    for package in apt libapt-pkg7.0; do \
        case "$package" in \
            apt) hash=0de729b8fe0e1c67c04e8e8d5bd6e0420dbd47f75b93f0b71c8e740edc2d5ffc; bytes=1506688 ;; \
            libapt-pkg7.0) hash=94e5d41303265b66af0f8c70ade18bfb77260ad286edee6c8045faddef82b94c; bytes=1139022 ;; \
            *) exit 2 ;; \
        esac; \
        test "$(wc -c < "/tmp/${package}.deb")" -eq "$bytes"; \
        printf '%s  %s\n' "$hash" "/tmp/${package}.deb" | sha256sum --check; \
        test "$(dpkg-deb -f "/tmp/${package}.deb" Package)" = "$package"; \
        test "$(dpkg-deb -f "/tmp/${package}.deb" Version)" = 3.2.1ubuntu0.1; \
        test "$(dpkg-deb -f "/tmp/${package}.deb" Architecture)" = amd64; \
        test "$(dpkg-query -W -f='${db:Status-Status}' "$package")" = installed; \
        test "$(dpkg-query -W -f='${Architecture}' "$package")" = amd64; \
        dpkg --compare-versions "$(dpkg-query -W -f='${Version}' "$package")" le 3.2.1ubuntu0.1; \
    done; \
    dpkg -i /tmp/libapt-pkg7.0.deb /tmp/apt.deb; \
    for package in apt libapt-pkg7.0; do \
        test "$(dpkg-query -W -f='${Version}' "$package")" = 3.2.1ubuntu0.1; \
        test "$(dpkg-query -W -f='${db:Status-Status}' "$package")" = installed; \
        rm -f "/tmp/${package}.deb"; \
    done; \
    dpkg-query -W -f='${binary:Package}\t${db:Status-Status}\n' | LC_ALL=C sort > "$after"; \
    cmp "$before" "$after"
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
RUN set -eux; \
    sed '/^libc.*=2.43-2ubuntu2.4$/d' /licenses/ubuntu-security-packages.lock > /tmp/runtime-security.lock; \
    UBUNTU_SECURITY_ARCHIVE=https://archive.ubuntu.com/ubuntu sh /usr/local/libexec/pasturestack/install-ubuntu-security-updates \
        /tmp/runtime-security.lock /etc/ssl/certs/ca-certificates.crt; \
    rm -f /tmp/runtime-security.lock
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
    for package in apt libapt-pkg7.0; do \
        test "$(dpkg-query -W -f='${Version}' "$package")" = 3.2.1ubuntu0.1; \
        test "$(dpkg-query -W -f='${db:Status-Status}' "$package")" = installed; \
    done; \
    dpkg-query -W -f='${binary:Package}\t${Version}\n' | LC_ALL=C sort > /usr/share/pasturestack/ubuntu-security-packages.tsv; \
    rm -f /var/cache/ldconfig/aux-cache

FROM scratch AS official_glibc
ADD --checksum=sha256:be83aefc14597fba683031a520f303950a7338b95d74decbe625faa51c49933e https://security.ubuntu.com/ubuntu/pool/main/g/glibc/libc6_2.43-2ubuntu2.7_amd64.deb /tmp/official-glibc/libc6.deb
ADD --checksum=sha256:c0393ea4fa3c45cefe0351db369d3a4a459edc8045856f2e3612ca17e77b2797 https://security.ubuntu.com/ubuntu/pool/main/g/glibc/libc-bin_2.43-2ubuntu2.7_amd64.deb /tmp/official-glibc/libc-bin.deb
ADD --checksum=sha256:408ba056d39651c2489b59522a614276d7a96020c6a401e73cd0da7d771764b0 https://security.ubuntu.com/ubuntu/pool/main/g/glibc/libc-gconv-modules-extra_2.43-2ubuntu2.7_amd64.deb /tmp/official-glibc/libc-gconv-modules-extra.deb
FROM qualified_backports AS runtime
RUN --mount=from=official_glibc,source=/tmp/official-glibc,target=/tmp/official-glibc,ro \
    set -eux; \
    for package in libc6 libc-bin libc-gconv-modules-extra; do \
        archive="/tmp/official-glibc/$package.deb"; \
        test "$(dpkg-deb -f "$archive" Package)" = "$package"; \
        test "$(dpkg-deb -f "$archive" Source)" = glibc; \
        test "$(dpkg-deb -f "$archive" Architecture)" = amd64; \
        test "$(dpkg-deb -f "$archive" Version)" = 2.43-2ubuntu2.7; \
        dpkg --compare-versions "$(dpkg-query -W -f='${Version}' "$package")" lt 2.43-2ubuntu2.7; \
    done; \
    dpkg --path-include='/usr/share/doc/libc6/*' --path-include='/usr/share/man/man1/getconf.1.gz' \
        --path-include='/usr/share/man/man1/tzselect.1.gz' --install /tmp/official-glibc/*.deb; \
    audit="$(dpkg --audit)"; \
    test -z "$audit"; \
    for package in libc6 libc-bin libc-gconv-modules-extra; do \
        integrity="$(dpkg --verify "$package")"; \
        test -z "$integrity"; \
    done; \
    getent hosts localhost >/dev/null; \
    rm -f /var/cache/ldconfig/aux-cache /var/log/dpkg.log; \
    dpkg-query -W -f='${binary:Package}\t${Version}\n' | LC_ALL=C sort > /usr/share/pasturestack/ubuntu-security-packages.tsv

CMD ["node-agent"]

#!/bin/bash
set -e

trap cleanup EXIT

cleanup()
{
    local exit=$?

    if [ -e $TEMP ]; then
        rm -rf $TEMP
    fi
    if [ -e $OLD ]; then
        rm -rf $OLD
    fi

    return $exit
}

check()
{
    local host_resolv="/etc/resolv.conf"
    local resolved_resolv="/run/systemd/resolve/resolv.conf"
    local resolved_content

    if grep -E -q '^nameserver 127.*.*.*|^nameserver localhost|^nameserver ::1' "${host_resolv}"; then
        if [ -e "${resolved_resolv}" ]; then
            resolved_content="$(cat "${resolved_resolv}")"
        elif [ -e /host/proc/1/ns/mnt ] && command -v nsenter >/dev/null 2>&1; then
            resolved_content="$(nsenter --mount=/host/proc/1/ns/mnt -- cat "${resolved_resolv}" 2>/dev/null || true)"
        fi

        if echo "${resolved_content}" | grep -E -q '^nameserver ' && \
           ! echo "${resolved_content}" | grep -E -q '^nameserver 127.*.*.*|^nameserver localhost|^nameserver ::1'; then
            info "DNS Checking detected systemd-resolved stub in ${host_resolv}; using ${resolved_resolv} as the real upstream DNS source"
            return 0
        fi

        error DNS Checking loopback IP address 127.0.0.0/8, localhost or ::1 configured as the DNS server on the host file /etc/resolv.conf, can’t accept it
        exit 1
    fi
}

PLATFORM_HOME=${PASTURESTACK_HOME:-${CATTLE_HOME:-/var/lib/pasturestack}}
source ${PLATFORM_HOME}/common/scripts.sh

DEST=$PLATFORM_HOME/node-agent
MAIN=$DEST/node-agent
STAMP=$PLATFORM_HOME/.node-agent-stamp
OLD=$(mktemp -d ${DEST}.XXXXXXXX)
TEMP=$(mktemp -d ${DEST}.XXXXXXXX)

cd $(dirname $0)

stage()
{
    if [[ -n "${CURL_CA_BUNDLE}" && -e ./dist/websocket/cacert.pem ]]; then
        if [ ! -e ./dist/websocket/cacert.orig ]; then
            cp ./dist/websocket/cacert.pem ./dist/websocket/cacert.orig
        fi
        cat ./dist/websocket/cacert.orig ${CURL_CA_BUNDLE} > ./dist/websocket/cacert.pem
    fi

    cp -rf apply.sh bin/node-agent $TEMP

    find $TEMP -name "*.sh" -exec chmod +x {} \;
    find $TEMP \( -name host-api -o -name cadvisor -o -name nsenter -o -name socat \) -exec chmod +x {} \;

    if [ -e $DEST ]; then
        mv $DEST ${OLD}
    fi
    mv $TEMP ${DEST}
    rm -rf ${OLD}

    echo $RANDOM > $STAMP

}

conf()
{
    CONF=(${PLATFORM_HOME}/node-agent/agent.conf
          ${CATTLE_HOME:-/var/lib/cattle}/pyagent/agent.conf
          /etc/cattle/agent/agent.conf
          ${CATTLE_HOME}/etc/cattle/agent/agent.conf
          /var/lib/rancher/etc/agent.conf)

    for conf_file in "${CONF[@]}"; do
        if [ -e $conf_file ]
        then
            source $conf_file
        fi
    done
}


start(){
    export PATH=${PLATFORM_HOME}/bin:$PATH

    if [ "${CATTLE_CHECK_NAMESERVER}" = "false" ];then
        info Skipping DNS nameserver check
    else
        check
    fi
    
    
    chmod +x $MAIN
    if [ "$CATTLE_PYPY" = "true" ] && which pypy >/dev/null; then
        MAIN="pypy $MAIN"
    fi

    info Executing $MAIN
    cleanup

    ${PLATFORM_HOME}/config.sh host-config

    exec $MAIN
}

conf

if [ "$1" = "start" ]; then
    start
else
    stage
fi

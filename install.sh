#!/bin/sh
set -e
set -o noglob

# Usage:
#   curl ... | ENV_VAR=... sh -
#       or
#   ENV_VAR=... ./install.sh
#
# Environment variables:
#   - RANCHERD_*
#     Environment variables which begin with RANCHERD_ will be preserved for the
#     systemd service to use. Setting RANCHERD_URL without explicitly setting
#     a systemd exec command will default the command to "agent", and we
#     enforce that RANCHERD_TOKEN or RANCHERD_CLUSTER_SECRET is also set.
#
#   - INSTALL_RANCHERD_SKIP_DOWNLOAD
#     If set to true will not download rancherd hash or binary.
#
#   - INSTALL_RANCHERD_FORCE_RESTART
#     If set to true will always restart the rancherd service
#
#   - INSTALL_RANCHERD_SKIP_ENABLE
#     If set to true will not enable or start rancherd service.
#
#   - INSTALL_RANCHERD_SKIP_START
#     If set to true will not start rancherd service.
#
#   - INSTALL_RANCHERD_VERSION
#     Version of rancherd to download from github. Will attempt to download from the
#     stable channel if not specified.
#
#   - INSTALL_RANCHERD_BIN_DIR
#     Directory to install rancherd binary, links, and uninstall script to, or use
#     /usr/local/bin as the default
#
#   - INSTALL_RANCHERD_BIN_DIR_READ_ONLY
#     If set to true will not write files to INSTALL_RANCHERD_BIN_DIR, forces
#     setting INSTALL_RANCHERD_SKIP_DOWNLOAD=true
#
#   - INSTALL_RANCHERD_SYSTEMD_DIR
#     Directory to install systemd service and environment files to, or use
#     /etc/systemd/system as the default
#
GITHUB_URL=https://github.com/rancher/rancherd/releases
DOWNLOADER=

# --- helper functions for logs ---
info()
{
    echo '[INFO] ' "$@"
}
warn()
{
    echo '[WARN] ' "$@" >&2
}
fatal()
{
    echo '[ERROR] ' "$@" >&2
    exit 1
}

# --- fatal if no systemd ---
verify_system() {
    if [ ! -d /run/systemd ]; then
        fatal 'Can not find systemd to use as a process supervisor for rancherd'
    fi
}

# --- add quotes to command arguments ---
quote() {
    for arg in "$@"; do
        printf '%s\n' "$arg" | sed "s/'/'\\\\''/g;1s/^/'/;\$s/\$/'/"
    done
}

# --- add indentation and trailing slash to quoted args ---
quote_indent() {
    printf ' \\\n'
    for arg in "$@"; do
        printf '\t%s \\\n' "$(quote "$arg")"
    done
}

# --- escape most punctuation characters, except quotes, forward slash, and space ---
escape() {
    printf '%s' "$@" | sed -e 's/\([][!#$%&()*;<=>?\_`{|}]\)/\\\1/g;'
}

# --- escape double quotes ---
escape_dq() {
    printf '%s' "$@" | sed -e 's/"/\\"/g'
}

# --- define needed environment variables ---
setup_env() {
    SYSTEM_NAME=rancherd

    # --- check for invalid characters in system name ---
    valid_chars=$(printf '%s' "${SYSTEM_NAME}" | sed -e 's/[][!#$%&()*;<=>?\_`{|}/[:space:]]/^/g;' )
    if [ "${SYSTEM_NAME}" != "${valid_chars}"  ]; then
        invalid_chars=$(printf '%s' "${valid_chars}" | sed -e 's/[^^]/ /g')
        fatal "Invalid characters for system name:
            ${SYSTEM_NAME}
            ${invalid_chars}"
    fi

    # --- use sudo if we are not already root ---
    SUDO=sudo
    if [ $(id -u) -eq 0 ]; then
        SUDO=
    fi

    # --- use binary install directory if defined or create default ---
    if [ -n "${INSTALL_RANCHERD_BIN_DIR}" ]; then
        BIN_DIR=${INSTALL_RANCHERD_BIN_DIR}
    else
        # --- use /usr/local/bin if root can write to it, otherwise use /opt/bin if it exists
        BIN_DIR=/usr/local/bin
        if ! $SUDO sh -c "touch ${BIN_DIR}/rancherd-ro-test && rm -rf ${BIN_DIR}/rancherd-ro-test"; then
            if [ -d /opt/bin ]; then
                BIN_DIR=/opt/bin
            fi
        fi
    fi

    # --- use systemd directory if defined or create default ---
    if [ -n "${INSTALL_RANCHERD_SYSTEMD_DIR}" ]; then
        SYSTEMD_DIR="${INSTALL_RANCHERD_SYSTEMD_DIR}"
    else
        SYSTEMD_DIR=/etc/systemd/system
    fi

    # --- set related files from system name ---
    SERVICE_RANCHERD=${SYSTEM_NAME}.service

    # --- use service or environment location depending on systemd ---
    FILE_RANCHERD_SERVICE=${SYSTEMD_DIR}/${SERVICE_RANCHERD}
    FILE_RANCHERD_ENV=${SYSTEMD_DIR}/${SERVICE_RANCHERD}.env

    # --- get hash of config & exec for currently installed rancherd ---
    PRE_INSTALL_HASHES=$(get_installed_hashes)

    # --- if bin directory is read only skip download ---
    if [ "${INSTALL_RANCHERD_BIN_DIR_READ_ONLY}" = true ]; then
        INSTALL_RANCHERD_SKIP_DOWNLOAD=true
    fi
}

# --- check if skip download environment variable set ---
can_skip_download() {
    if [ "${INSTALL_RANCHERD_SKIP_DOWNLOAD}" != true ]; then
        return 1
    fi
}

# --- verify an executable rancherd binary is installed ---
verify_rancherd_is_executable() {
    if [ ! -x ${BIN_DIR}/rancherd ]; then
        fatal "Executable rancherd binary not found at ${BIN_DIR}/rancherd"
    fi
}

# --- set arch and suffix, fatal if architecture not supported ---
setup_verify_arch() {
    if [ -z "$ARCH" ]; then
        ARCH=$(uname -m)
    fi
    case $ARCH in
        amd64)
            ARCH=amd64
            SUFFIX=-${ARCH}
            ;;
        x86_64)
            ARCH=amd64
            SUFFIX=-amd64
            ;;
        #arm64)
        #    ARCH=arm64
        #    SUFFIX=-${ARCH}
        #    ;;
        #aarch64)
        #    ARCH=arm64
        #    SUFFIX=-${ARCH}
        #    ;;
        #arm*)
        #    ARCH=arm
        #    SUFFIX=-${ARCH}hf
        #    ;;
        *)
            fatal "Unsupported architecture $ARCH"
    esac
}

# --- verify existence of network downloader executable ---
verify_downloader() {
    # Return failure if it doesn't exist or is no executable
    [ -x "$(command -v $1)" ] || return 1

    # Set verified executable as our downloader program and return success
    DOWNLOADER=$1
    return 0
}

# --- create temporary directory and cleanup when done ---
setup_tmp() {
    TMP_DIR=$(mktemp -d -t rancherd-install.XXXXXXXXXX)
    TMP_HASH=${TMP_DIR}/rancherd.hash
    TMP_BIN=${TMP_DIR}/rancherd.bin
    cleanup() {
        code=$?
        set +e
        trap - EXIT
        rm -rf ${TMP_DIR}
        exit $code
    }
    trap cleanup INT EXIT
}

# --- use desired rancherd version if defined or find version from channel ---
get_release_version() {
    if [ -n "${INSTALL_RANCHERD_VERSION}" ]; then
        VERSION_RANCHERD=${INSTALL_RANCHERD_VERSION}
    else
        version_url="${GITHUB_URL}/latest"
        case $DOWNLOADER in
            curl)
                VERSION_RANCHERD=$(curl -w '%{url_effective}' -L -s -S ${version_url} -o /dev/null | sed -e 's|.*/||')
                ;;
            wget)
                VERSION_RANCHERD=$(wget -SqO /dev/null ${version_url} 2>&1 | grep -i Location | sed -e 's|.*/||')
                ;;
            *)
                fatal "Incorrect downloader executable '$DOWNLOADER'"
                ;;
        esac
    fi
    info "Using ${VERSION_RANCHERD} as release"
}

# --- download from github url ---
download() {
    [ $# -eq 2 ] || fatal 'download needs exactly 2 arguments'

    case $DOWNLOADER in
        curl)
            curl -o $1 -sfL $2
            ;;
        wget)
            wget -qO $1 $2
            ;;
        *)
            fatal "Incorrect executable '$DOWNLOADER'"
            ;;
    esac

    # Abort if download command failed
    [ $? -eq 0 ] || fatal 'Download failed'
}

# --- download hash from github url ---
download_hash() {
    HASH_URL=${GITHUB_URL}/download/${VERSION_RANCHERD}/sha256sum-${ARCH}.txt
    info "Downloading hash ${HASH_URL}"
    download ${TMP_HASH} ${HASH_URL}
    HASH_EXPECTED=$(grep " rancherd${SUFFIX}$" ${TMP_HASH})
    HASH_EXPECTED=${HASH_EXPECTED%%[[:blank:]]*}
}

# --- check hash against installed version ---
installed_hash_matches() {
    if [ -x ${BIN_DIR}/rancherd ]; then
        HASH_INSTALLED=$(sha256sum ${BIN_DIR}/rancherd)
        HASH_INSTALLED=${HASH_INSTALLED%%[[:blank:]]*}
        if [ "${HASH_EXPECTED}" = "${HASH_INSTALLED}" ]; then
            return
        fi
    fi
    return 1
}

# --- download binary from github url ---
download_binary() {
    BIN_URL=${GITHUB_URL}/download/${VERSION_RANCHERD}/rancherd${SUFFIX}
    info "Downloading binary ${BIN_URL}"
    download ${TMP_BIN} ${BIN_URL}
}

# --- verify downloaded binary hash ---
verify_binary() {
    info "Verifying binary download"
    HASH_BIN=$(sha256sum ${TMP_BIN})
    HASH_BIN=${HASH_BIN%%[[:blank:]]*}
    if [ "${HASH_EXPECTED}" != "${HASH_BIN}" ]; then
        fatal "Download sha256 does not match ${HASH_EXPECTED}, got ${HASH_BIN}"
    fi
}

# --- setup permissions and move binary to system directory ---
setup_binary() {
    chmod 755 ${TMP_BIN}
    info "Installing rancherd to ${BIN_DIR}/rancherd"
    $SUDO chown root:root ${TMP_BIN}
    $SUDO mv -f ${TMP_BIN} ${BIN_DIR}/rancherd
}

# --- download and verify rancherd ---
download_and_verify() {
    if can_skip_download; then
       info 'Skipping rancherd download and verify'
       verify_rancherd_is_executable
       return
    fi

    setup_verify_arch
    verify_downloader curl || verify_downloader wget || fatal 'Can not find curl or wget for downloading files'
    setup_tmp
    get_release_version
    download_hash

    if installed_hash_matches; then
        info 'Skipping binary downloaded, installed rancherd matches hash'
        return
    fi

    download_binary
    verify_binary
    setup_binary
}

# --- disable current service if loaded --
systemd_disable() {
    $SUDO systemctl disable ${SYSTEM_NAME} >/dev/null 2>&1 || true
    $SUDO rm -f /etc/systemd/system/${SERVICE_RANCHERD} || true
    $SUDO rm -f /etc/systemd/system/${SERVICE_RANCHERD}.env || true
}

# --- capture current env and create file containing rancherd_ variables ---
create_env_file() {
    info "env: Creating environment file ${FILE_RANCHERD_ENV}"
    $SUDO touch ${FILE_RANCHERD_ENV}
    $SUDO chmod 0600 ${FILE_RANCHERD_ENV}
    env | grep '^RANCHERD_' | $SUDO tee ${FILE_RANCHERD_ENV} >/dev/null
    env | grep -Ei '^(NO|HTTP|HTTPS)_PROXY' | $SUDO tee -a ${FILE_RANCHERD_ENV} >/dev/null
}

# --- write systemd service file ---
create_systemd_service_file() {
    info "systemd: Creating service file ${FILE_RANCHERD_SERVICE}"
    $SUDO tee ${FILE_RANCHERD_SERVICE} >/dev/null << EOF
[Unit]
Description=Rancher Bootstrap
Documentation=https://github.com/rancher/rancherd
Wants=network-online.target
After=network-online.target

[Install]
WantedBy=multi-user.target

[Service]
Type=oneshot
EnvironmentFile=-/etc/default/%N
EnvironmentFile=-/etc/sysconfig/%N
EnvironmentFile=-${FILE_RANCHERD_ENV}
KillMode=process
# Having non-zero Limit*s causes performance problems due to accounting overhead
# in the kernel. We recommend using cgroups to do container-local accounting.
LimitNOFILE=1048576
LimitNPROC=infinity
LimitCORE=infinity
TasksMax=infinity
TimeoutStartSec=0
ExecStart=${BIN_DIR}/rancherd bootstrap
EOF
}

# --- get hashes of the current rancherd bin and service files
get_installed_hashes() {
    $SUDO sha256sum ${BIN_DIR}/rancherd ${FILE_RANCHERD_SERVICE} ${FILE_RANCHERD_ENV} 2>&1 || true
}

# --- enable and start systemd service ---
systemd_enable() {
    info "systemd: Enabling ${SYSTEM_NAME} unit"
    $SUDO systemctl enable ${FILE_RANCHERD_SERVICE} >/dev/null
    $SUDO systemctl daemon-reload >/dev/null
}

systemd_start() {
    info "systemd: Starting ${SYSTEM_NAME}"
    $SUDO systemctl restart --no-block ${SYSTEM_NAME}
    info "Run \"journalctl -u ${SYSTEM_NAME} -f\" to watch logs"
}

# --- enable and start openrc service ---
openrc_enable() {
    info "openrc: Enabling ${SYSTEM_NAME} service for default runlevel"
    $SUDO rc-update add ${SYSTEM_NAME} default >/dev/null
}

openrc_start() {
    info "openrc: Starting ${SYSTEM_NAME}"
    $SUDO ${FILE_RANCHERD_SERVICE} restart
}

# --- startup systemd or openrc service ---
service_enable_and_start() {
    [ "${INSTALL_RANCHERD_SKIP_ENABLE}" = true ] && return

    systemd_enable

    [ "${INSTALL_RANCHERD_SKIP_START}" = true ] && return

    POST_INSTALL_HASHES=$(get_installed_hashes)
    if [ "${PRE_INSTALL_HASHES}" = "${POST_INSTALL_HASHES}" ] && [ "${INSTALL_RANCHERD_FORCE_RESTART}" != true ]; then
        info 'No change detected so skipping service start'
        return
    fi

    systemd_start
    return 0
}

# --- re-evaluate args to include env command ---
eval set -- $(escape "${INSTALL_RANCHERD_EXEC}") $(quote "$@")

# --- run the install process --
{
    verify_system
    setup_env "$@"
    download_and_verify
    systemd_disable
    create_env_file
    create_systemd_service_file
    service_enable_and_start
}

#!/bin/bash
# install-rocm-tarball.sh — download and install ROCm from an S3 therock nightly tarball.
#
# Usage: install-rocm-tarball.sh <ROCM_VERSION> <ROCM_TARBALL_URL> [<LIBDRM_SYMLINK_DIR>]
#
#   ROCM_VERSION       — version string, e.g. "7.12"
#   ROCM_TARBALL_URL   — full URL to the .tar.gz tarball
#   LIBDRM_SYMLINK_DIR — directory to create libdrm_amdgpu.so.1 symlink in
#                        (default: /opt/rocm/lib)
#
# Integrity: TheRock nightly tarballs are served over HTTPS from S3 and do not
# have companion checksum files. The tarball URL encodes the build date (e.g.
# 7.12.0a20260225), providing implicit version pinning.
#
# After install:
#   /opt/rocm-<version>/  — extracted tarball
#   /etc/alternatives/rocm -> /opt/rocm-<version>
#   /opt/rocm             -> /etc/alternatives/rocm
#   <LIBDRM_SYMLINK_DIR>/libdrm_amdgpu.so{,.1} -> rocm_sysdeps/lib/libdrm_amdgpu.so
#   <LIBDRM_SYMLINK_DIR>/libdrm.so{,.2}        -> rocm_sysdeps/lib/libdrm.so  (if present)

set -euo pipefail

ROCM_VERSION="${1:?ROCM_VERSION required}"
ROCM_TARBALL_URL="${2:?ROCM_TARBALL_URL required}"
LIBDRM_SYMLINK_DIR="${3:-/opt/rocm/lib}"

echo "=== ROCm install: S3 tarball (${ROCM_VERSION}) ==="
echo "    libdrm symlink dir: ${LIBDRM_SYMLINK_DIR}"

mkdir -p "/opt/rocm-${ROCM_VERSION}"
# arg 2 may be a local file path (bind-mounted, preferred — downloaded
# once on the host) or a URL (legacy fallback). Extract from the local file when
# it exists; otherwise download.
if [ -f "${ROCM_TARBALL_URL}" ]; then
    echo "    source: local file (bind-mounted, no download)"
    tar -xzf "${ROCM_TARBALL_URL}" -C "/opt/rocm-${ROCM_VERSION}"
else
    echo "    source: URL (download)"
    wget -qO- "${ROCM_TARBALL_URL}" | tar -xzf - -C "/opt/rocm-${ROCM_VERSION}"
fi

# Prune unneeded content to reduce image size.
# Static libs, headers, cmake/pkgconfig, docs, tests, benchmarks and sample
# data are not required at container runtime. Removing them in the same RUN
# layer as the extract keeps the final Docker layer as small as possible.
ROCM_DIR="/opt/rocm-${ROCM_VERSION}"

# Keep only what is needed at runtime for DCM (amd-smi only):
#   bin/amd-smi, libexec/amdsmi_cli/, share/amd_smi/amdsmi/
#   libamd_smi.so (needed by amdsmi Python wrapper), rocm_sysdeps
# Stage needed content into a temp dir, wipe ROCM_DIR, restore only those.
echo "=== Pruning unreferenced libs from ${ROCM_DIR} ==="
KEEP_DIR=$(mktemp -d)
mkdir -p "${KEEP_DIR}/lib" "${KEEP_DIR}/bin" "${KEEP_DIR}/libexec" "${KEEP_DIR}/share"
# Save rocm_sysdeps (libdrm symlinks) inside lib/
[ -d "${ROCM_DIR}/lib/rocm_sysdeps" ] && \
    cp -a "${ROCM_DIR}/lib/rocm_sysdeps" "${KEEP_DIR}/lib/rocm_sysdeps" || true
# Save libamd_smi into lib/ (needed by amdsmi Python wrapper)
for f in "${ROCM_DIR}/lib"/libamd_smi.*; do
    if [ -e "$f" ] || [ -L "$f" ]; then
        cp -a "$f" "${KEEP_DIR}/lib/" 2>/dev/null || true
    fi
done
# Save amd-smi CLI and its Python modules
[ -f "${ROCM_DIR}/bin/amd-smi" ] && \
    cp -a "${ROCM_DIR}/bin/amd-smi" "${KEEP_DIR}/bin/" || true
[ -d "${ROCM_DIR}/libexec/amdsmi_cli" ] && \
    cp -a "${ROCM_DIR}/libexec/amdsmi_cli" "${KEEP_DIR}/libexec/" || true
[ -d "${ROCM_DIR}/share/amd_smi" ] && \
    cp -a "${ROCM_DIR}/share/amd_smi" "${KEEP_DIR}/share/" || true
# Wipe entire ROCM_DIR and rebuild minimal structure
if [ -z "${ROCM_DIR}" ]; then
    echo "Refusing to delete empty ROCM_DIR" >&2
    exit 1
fi
case "${ROCM_DIR}" in
    /opt/rocm-*) ;;
    *)
        echo "Refusing to delete unexpected ROCM_DIR: ${ROCM_DIR}" >&2
        exit 1
        ;;
esac
rm -rf "${ROCM_DIR}"
mkdir -p "${ROCM_DIR}"
cp -a "${KEEP_DIR}/." "${ROCM_DIR}/"
rm -rf "${KEEP_DIR}"

echo "=== Pruning done ==="

mkdir -p /etc/alternatives
ln -sf "/opt/rocm-${ROCM_VERSION}" /etc/alternatives/rocm
ln -sf /etc/alternatives/rocm /opt/rocm

SYSDEPS="/opt/rocm-${ROCM_VERSION}/lib/rocm_sysdeps/lib"
mkdir -p "${LIBDRM_SYMLINK_DIR}"

if [ -f "${SYSDEPS}/libdrm_amdgpu.so" ]; then
    ln -sf "${SYSDEPS}/libdrm_amdgpu.so" "${LIBDRM_SYMLINK_DIR}/libdrm_amdgpu.so.1"
    ln -sf "${SYSDEPS}/libdrm_amdgpu.so" "${LIBDRM_SYMLINK_DIR}/libdrm_amdgpu.so"
fi

if [ -f "${SYSDEPS}/libdrm.so" ]; then
    ln -sf "${SYSDEPS}/libdrm.so" "${LIBDRM_SYMLINK_DIR}/libdrm.so.2"
    ln -sf "${SYSDEPS}/libdrm.so" "${LIBDRM_SYMLINK_DIR}/libdrm.so"
fi

echo "=== ROCm ${ROCM_VERSION} installed at /opt/rocm-${ROCM_VERSION} ==="

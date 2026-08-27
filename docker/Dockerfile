ARG RHEL_BASE_MIN_IMAGE=registry.access.redhat.com/ubi9/ubi-minimal:9.8
FROM ${RHEL_BASE_MIN_IMAGE}

LABEL name="device-config-manager" \
    maintainer="nikhil.komarla@amd.com,sriram.ravishankar@amd.com,shrey.ajmera@amd.com,yan.sun3@amd.com" \
    vendor="Advanced Micro Devices, Inc." \
    version="latest" \
    release="latest" \
    OS="registry.access.redhat.com/ubi9/ubi-minimal:9.8" \
    summary="Device Config Manager (DCM) manages AMD device configurations, starting with GPU partitioning, and uses a Kubernetes config-map associated with a DCM daemonset." \
    description="Device Config Manager (DCM) manages AMD device configurations, users provide the necessary GPU configurations through a Kubernetes config-map, which is then associated with the DCM daemonset to ensure proper application and management of these configurations across the cluster."

# unused for now revisit
ARG ROCM_VERSION=10.0.0rc4
ARG AMDGPU_VERSION=10.0.0rc4
# S3 therock nightly tarball (optional). Set ROCM_TARBALL_URL to install ROCm
# from a therock tarball instead of repo.radeon.com.
ARG ROCM_TARBALL_URL=""
# Whether the LD_PRELOADed libs in ./smilib were staged from the therock tarball
# (1) or the committed assets/ (0). The Makefile stages docker/smilib from
# build/assets in both modes, so the ADD below is identical either way; this ARG
# records the mode for traceability.
ARG AMDSMI_FROM_TARBALL=1

ENV PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/home/amd/bin/:/opt/rocm/bin/
# /home/amd/lib holds the LD_PRELOADed libamd_smi (so.27, ROCm 10.0) + its rocm_sysdeps netlink
# DT_NEEDEDs (mnl, nl_3, nl_genl_3). /opt/rocm-${ROCM_VERSION}/lib/rocm_sysdeps/lib is
# needed for therock tarball builds (librocm_sysdeps_drm.so.2 etc.); for repo.radeon.com
# builds it is a no-op.
ENV LD_LIBRARY_PATH=/home/amd/lib:/opt/rocm/lib:/opt/rocm-${ROCM_VERSION}/lib/rocm_sysdeps/lib:$LD_LIBRARY_PATH

ADD rhelrepofiles/amdgpu.repo /etc/yum.repos.d/amdgpu.repo
ADD rhelrepofiles/rocm.repo /etc/yum.repos.d/rocm.repo
ADD install-rocm-tarball.sh /usr/local/bin/install-rocm-tarball.sh
RUN chmod +x /usr/local/bin/install-rocm-tarball.sh

# ROCM_TARBALL_URL is now only a MODE SELECTOR (non-empty = tarball
# mode); the tarball data comes from the bind-mounted `rocm-tarball` context, so
# install-rocm-tarball.sh reads a local file and no wget is needed.
RUN --mount=type=bind,from=rocm-tarball,source=therock.tar.gz,target=/tmp/rocm.tar.gz \
   microdnf clean all && \
   microdnf update -y && \
   if [ -n "${ROCM_TARBALL_URL}" ]; then \
      microdnf install -y tar gzip findutils sudo procps file libcap-devel vim net-tools kmod \
         iptables-libs dbus dbus-tools pciutils jq python3 && \
      /usr/local/bin/install-rocm-tarball.sh "${ROCM_VERSION}" /tmp/rocm.tar.gz /usr/lib64; \
   else \
      microdnf install -y sudo procps file libcap-devel libdrm-amdgpu vim net-tools kmod \
         iptables-libs dbus dbus-tools pciutils jq && \
      microdnf install -y amd-smi-lib; \
   fi && \
   rm -rf /var/cache/yum && rm -rf /var/cache/dnf && microdnf clean all
RUN mkdir -p /home/amd/bin/
RUN mkdir -p /home/amd/lib/

ADD ./smilib/ /home/amd/lib/
ADD ./device-config-manager /home/amd/bin/server
RUN mkdir -p /home/amd/tools/

ADD ./entrypoint.sh /home/amd/tools/entrypoint.sh
RUN chmod +x /home/amd/tools/entrypoint.sh

ADD LICENSE /licenses/LICENSE

ENTRYPOINT ["/home/amd/tools/entrypoint.sh"]

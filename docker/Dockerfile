FROM registry.access.redhat.com/ubi9/ubi-minimal:9.7

LABEL name="device-config-manager" \
    maintainer="nikhil.komarla@amd.com,sriram.ravishankar@amd.com,shrey.ajmera@amd.com,yan.sun3@amd.com" \
    vendor="Advanced Micro Devices, Inc." \
    version="latest" \
    release="latest" \
    OS="registry.access.redhat.com/ubi9/ubi-minimal:9.7" \
    summary="Device Config Manager (DCM) manages AMD device configurations, starting with GPU partitioning, and uses a Kubernetes config-map associated with a DCM daemonset." \
    description="Device Config Manager (DCM) manages AMD device configurations, users provide the necessary GPU configurations through a Kubernetes config-map, which is then associated with the DCM daemonset to ensure proper application and management of these configurations across the cluster."

# unused for now revisit
ARG ROCM_VERSION=7.2.1
ARG AMDGPU_VERSION=7.2.1
# S3 therock nightly tarball (optional). Set ROCM_TARBALL_URL to install ROCm
# from a therock tarball instead of repo.radeon.com.
ARG ROCM_TARBALL_URL=""

ENV PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/home/amd/bin/:/opt/rocm/bin/
# /opt/rocm-${ROCM_VERSION}/lib/rocm_sysdeps/lib is needed for therock tarball builds
# (librocm_sysdeps_drm.so.2 etc.). For repo.radeon.com builds it is a no-op.
ENV LD_LIBRARY_PATH=/opt/rocm/lib:/opt/rocm-${ROCM_VERSION}/lib/rocm_sysdeps/lib:$LD_LIBRARY_PATH

ADD rhelrepofiles/amdgpu.repo /etc/yum.repos.d/amdgpu.repo
ADD rhelrepofiles/rocm.repo /etc/yum.repos.d/rocm.repo
ADD install-rocm-tarball.sh /usr/local/bin/install-rocm-tarball.sh
RUN chmod +x /usr/local/bin/install-rocm-tarball.sh

RUN microdnf clean all && \
   microdnf update -y && \
   if [ -n "${ROCM_TARBALL_URL}" ]; then \
      microdnf install -y wget tar gzip findutils sudo procps file libcap-devel vim net-tools kmod \
         iptables-libs dbus dbus-tools pciutils jq python3 && \
      /usr/local/bin/install-rocm-tarball.sh "${ROCM_VERSION}" "${ROCM_TARBALL_URL}" /usr/lib64; \
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

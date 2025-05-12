FROM registry.access.redhat.com/ubi9/ubi-minimal:9.4

LABEL name="device-config-manager" \ 
    maintainer="nikhil.komarla@amd.com,sriram.ravishankar@amd.com,shrey.ajmera@amd.com,yan.sun3@amd.com" \
    vendor="Advanced Micro Devices, Inc." \
    version="v1.3.1" \
    release="v1.3.1" \
    OS="registry.access.redhat.com/ubi9/ubi-minimal:9.4" \
    summary="Device Config Manager (DCM) manages AMD device configurations, starting with GPU partitioning, and uses a Kubernetes config-map associated with a DCM daemonset." \
    description="Device Config Manager (DCM) manages AMD device configurations, users provide the necessary GPU configurations through a Kubernetes config-map, which is then associated with the DCM daemonset to ensure proper application and management of these configurations across the cluster."

# unused for now revisit
ARG ROCM_VERSION=6.4
ARG AMDGPU_VERSION=6.4

ADD rhelrepofiles/amdgpu.repo /etc/yum.repos.d/amdgpu.repo 
ADD rhelrepofiles/rocm.repo /etc/yum.repos.d/rocm.repo

RUN microdnf clean all && microdnf install -y sudo procps file libcap-devel libdrm-amdgpu vim net-tools kmod && microdnf install -y amd-smi-lib && rm -rf /var/cache/yum && rm -rf /var/cache/dnf && microdnf clean all

ENV PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/home/amd/bin/:/opt/rocm/bin/
RUN mkdir -p /home/amd/bin/

ENV LD_LIBRARY_PATH=/opt/rocm/lib
ADD ./device-config-manager /home/amd/bin/server
RUN mkdir -p /home/amd/tools/

ADD ./entrypoint.sh /home/amd/tools/entrypoint.sh
RUN chmod +x /home/amd/tools/entrypoint.sh

ADD LICENSE /licenses/LICENSE

ENTRYPOINT ["/home/amd/tools/entrypoint.sh"]

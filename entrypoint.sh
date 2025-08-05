#!/usr/bin/env bash
set -x
set -euo pipefail
dir=/usr/src/github.com/ROCm/device-config-manager
netns=/var/run/netns
dockerdir=/etc/docker

term() {
    killall dockerd
    wait
}

PATH=/usr/local/go/bin:$PATH

mkdir -p ${dockerdir}
echo 'DOCKER_OPTS="--config-file=/etc/docker/daemon.json"' >> /etc/default/docker
echo '{"insecure-registries" : ["registry.test.pensando.io:5000"]}' > ${dockerdir}/daemon.json

dockerd -s vfs &

trap term INT TERM

mkdir -p ${dir}
mkdir -p ${netns}
mount -o bind /device-config-manager ${dir}
rm -f $dir/.container_ready
export GOFLAGS=-mod=vendor
sysctl -w vm.max_map_count=262144

touch $dir/.container_ready
exec "$@"
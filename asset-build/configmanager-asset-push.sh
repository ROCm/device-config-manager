#!/bin/bash

if [ -z $RELEASE ]
then
  echo "RELEASE is not set, return"

  if [ -z ${DOCKERHUB_TOKEN-} ]
  then
      echo "DOCKERHUB_TOKEN is not set"
  else
      echo "DOCKERHUB_TOKEN is set"
  fi

  exit 0
fi

tag_prefix="${RELEASE%-*}"

if [ "$tag_prefix" == "config-manager-0.0.1" ]; then
  tag="latest"
else
  tag="$tag_prefix"
fi

echo "Copying device-config-manager artifacts..."

setup_dir () {
    ls -al /device-config-manager/
    BUNDLE_DIR=/device-config-manager/output/
    mkdir -p $BUNDLE_DIR
}

copy_artifacts () {
    # remove 'configmanager-' from release label for upstream version changes
    DEBIAN_VERSION="${RELEASE:9}"
    # copy device-config-manager binary
    cp /device-config-manager/bin/device-config-manager $BUNDLE_DIR/device-config-manager-$RELEASE.gobin
    # copy docker image
    cp /device-config-manager/docker/obj/config-manager-ubi-latest.tgz $BUNDLE_DIR/device-config-manager-$RELEASE.tar.gz
    # copy device-config-manager debian
    cp /device-config-manager/bin/amdgpu-configmanager_22.04_amd64.deb $BUNDLE_DIR/amdgpu-configmanager_${DEBIAN_VERSION}~22.04_amd64.deb
    # copy device-config-manager debian 24.04
    cp /device-config-manager/bin/amdgpu-configmanager_24.04_amd64.deb $BUNDLE_DIR/amdgpu-configmanager_${DEBIAN_VERSION}~24.04_amd64.deb
    # copy helm-charts
    cp /device-config-manager/helm-charts/device-config-manager-charts-v1.0.0.tgz $BUNDLE_DIR/device-config-manager-charts-$RELEASE-v1.0.0.tgz
    # list the artifacts copied out
    ls -la $BUNDLE_DIR
}

docker_push () {
    CONFIG_MANAGER_IMAGE_URL=registry.test.pensando.io:5000/device-config-manager

    # rhel 9.4 image push
    docker load -i /device-config-manager/docker/obj/config-manager-ubi9-latest.tgz
    docker inspect $CONFIG_MANAGER_IMAGE_URL:latest | grep "HOURLY"
    docker tag $CONFIG_MANAGER_IMAGE_URL:latest $CONFIG_MANAGER_IMAGE_URL:$tag
    docker push $CONFIG_MANAGER_IMAGE_URL:$tag

    if [ -z $DOCKERHUB_TOKEN ]
    then
      echo "DOCKERHUB_TOKEN is not set"
    else
      # rhel 9.4
      docker login --username=shreyajmeraamd --password-stdin <<< $DOCKERHUB_TOKEN
      docker tag $CONFIG_MANAGER_IMAGE_URL:$tag amdpsdo/device-config-manager:$RELEASE
      docker push amdpsdo/device-config-manager:$RELEASE

    fi
}

setup () {
    setup_dir
    copy_artifacts
    docker_push
}

upload () {
    cd $BUNDLE_DIR
    find . -type f -print0 | while IFS= read -r -d $'\0' file;
      do asset-push builds hourly-device-config-manager $RELEASE "$file" ;
      if [ $? -ne 0 ]; then
        exit 1
      fi
    done
}

main () {
  setup
  upload
}

main
exit 0

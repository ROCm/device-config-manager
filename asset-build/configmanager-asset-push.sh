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
    DEBIAN_VERSION="${RELEASE:1}"
    # copy docker image
    cp /device-config-manager/docker/obj/config-manager-ubi22-latest.tgz $BUNDLE_DIR/device-config-manager-$RELEASE.tar.gz
    # copy docker image ubi24
    cp /device-config-manager/docker/obj/config-manager-ubi24-latest.tgz $BUNDLE_DIR/device-config-manager-$RELEASE-noble.tar.gz
    # copy helm-charts
    cp /device-config-manager/helm-charts/device-config-manager-charts-v1.3.0.tgz $BUNDLE_DIR/device-config-manager-charts-$RELEASE.tgz
    # list the artifacts copied out
    ls -la $BUNDLE_DIR
}

docker_push () {
    CONFIG_MANAGER_IMAGE_URL=registry.test.pensando.io:5000/device-config-manager

    # rhel 9.4 image push
    docker load -i /device-config-manager/docker/obj/config-manager-ubi22-latest.tgz
    docker inspect $CONFIG_MANAGER_IMAGE_URL:latest | grep "HOURLY"
    docker tag $CONFIG_MANAGER_IMAGE_URL:latest $CONFIG_MANAGER_IMAGE_URL:$tag
    docker push $CONFIG_MANAGER_IMAGE_URL:$tag

    # rhel 9.4 image push ubi 24
    $ubi24_tag = "$tag-ubi24"
    docker load -i /device-config-manager/docker/obj/config-manager-ubi24-latest.tgz
    docker inspect $CONFIG_MANAGER_IMAGE_URL:latest | grep "HOURLY"
    docker tag $CONFIG_MANAGER_IMAGE_URL:latest $CONFIG_MANAGER_IMAGE_URL:$ubi24_tag
    docker push $CONFIG_MANAGER_IMAGE_URL:$ubi24_tag

    if [ -z $DOCKERHUB_TOKEN ]
    then
      echo "DOCKERHUB_TOKEN is not set"
    else
      # rhel 9.4
      docker login --username=shreyajmeraamd --password-stdin <<< $DOCKERHUB_TOKEN
      docker tag $CONFIG_MANAGER_IMAGE_URL:$tag amdpsdo/device-config-manager:$RELEASE
      docker push amdpsdo/device-config-manager:$RELEASE

      docker tag $CONFIG_MANAGER_IMAGE_URL:$ubi24_tag amdpsdo/device-config-manager:$RELEASE-ubi24
      docker push amdpsdo/device-config-manager:$RELEASE-ubi24

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

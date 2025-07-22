#!/bin/bash -e
#
# Copyright(C) Advanced Micro Devices, Inc. All rights reserved.
#
# You may not use this software and documentation (if any) (collectively,
# the "Materials") except in compliance with the terms and conditions of
# the Software License Agreement included with the Materials or otherwise as
# set forth in writing and signed by you and an authorized signatory of AMD.
# If you do not have a copy of the Software License Agreement, contact your
# AMD representative for a copy.
#
# You agree that you will not reverse engineer or decompile the Materials,
# in whole or in part, except as allowed by applicable law.
#
# THE MATERIALS ARE DISTRIBUTED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OR
# REPRESENTATIONS OF ANY KIND, EITHER EXPRESS OR IMPLIED.
#
#
# script to generate tarball with config manager docker, entrypoint script
# and docker_run script to create the docker container

# ./script.sh -h                # Calls print_help and exits
# ./script.sh -s                # Prints "saving image option set" and sets SAVE_IMAGE=1
# ./script.sh -n my_image       # Sets DOCKER_IMAGE_NAME="my_image"
# ./script.sh -p                # Prints "publish image option set" and sets PUBLISH_IMAGE=1
# ./script.sh -n my_image -s -p # Combines options: sets DOCKER_IMAGE_NAME, SAVE_IMAGE, and PUBLISH_IMAGE

print_help () {
    echo "This script can be used to build a exporter container"
    echo
    echo "Syntax: $0 [-s -n]"
    echo "options:"
    echo "-h    print help"
    echo "-s    prepare a release tarball image"
    echo "-p    publish to registry"
    echo "-n    docker image name"
    exit 0
}

while getopts "hsn:p" option; do
    case $option in
        h)
            print_help
            exit ;;
        s)
            echo "saving image option set"
            SAVE_IMAGE=1
            ;;
        n)
            DOCKER_IMAGE_NAME=$OPTARG ;;
        p)
            echo "publish image option set"
            PUBLISH_IMAGE=1
            ;;
        \?)
            echo "Error: Invalid argument"
            exit ;;
    esac
done

VER=v1
if [ -z $RELEASE ]; then
  echo "RELEASE is not set, return"
else
  tag_prefix="${RELEASE%-*}"

  if [ "$tag_prefix" == "config-manager-0.0.1" ]; then
    VER="latest"
  else
    VER="$tag_prefix"
  fi
fi

IMAGE_DIR=$(pwd)/obj

rm -rf $IMAGE_DIR
mkdir -p $IMAGE_DIR

DOCKER_REGISTRY="registry.test.pensando.io:5000/device-config-manager"
IMAGE_URL="${DOCKER_REGISTRY}:${VER}"

echo $TOP_DIR
rm -rf $TOP_DIR/docker/smilib $TOP_DIR/docker/device-config-manager
sleep 5

# Always use RHEL9 OS for both openshift and K8s env
cp -r $TOP_DIR/assets/amd_smi_lib/x86_64/RHEL9/lib $TOP_DIR/docker/smilib
cp $TOP_DIR/bin/device-config-manager-$UBUNTU_VERSION $TOP_DIR/docker/device-config-manager

if [ $PUBLISH_IMAGE == 1 ]; then
    echo "publishing dcm image to $IMAGE_URL"
    docker build -t $IMAGE_URL . --label HOURLY_TAG=$HOURLY_TAG_LABEL -f Dockerfile && docker push $IMAGE_URL
    if [ $? -eq 0 ]; then
        echo "Successfully published image $IMAGE_URL"
    else
        echo "Failed to publish docker image"
        exit $?
    fi
else
    echo "building dcm image to $DOCKER_IMAGE_NAME"
    docker build -t $IMAGE_URL . --label HOURLY_TAG=$HOURLY_TAG_LABEL -f Dockerfile && docker save -o config-manager-$VER.tar $IMAGE_URL
    if [ $? -eq 0 ]; then
        gzip config-manager-$VER.tar
        mv config-manager-$VER.tar.gz config-manager-$VER.tgz
    else
        echo "Failed to build docker image"
        exit $?
    fi
fi

# prepare the final tar ball now
if [ "$SAVE_IMAGE" == 1 ]; then
    echo "Preparing final image ..."
    mv config-manager-$VER.tgz $IMAGE_DIR/config-manager-ubi9-latest.tgz
    echo "Image ready in $IMAGE_DIR"
fi

rm -rf $TOP_DIR/docker/smilib $TOP_DIR/docker/device-config-manager

exit 0

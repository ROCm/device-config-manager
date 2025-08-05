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

print_help() {
    echo "Usage: entrypoint.sh [options]"
    echo
    echo "Options:"
    echo "-h    Show this help message"
    echo "-k    Trigger DCM Build for K8S"
    echo "-d    Trigger DCM Build for debian"
}

while getopts "hdk" option; do
    case $option in
        h)
            print_help
            exit 0 ;;
        d)
            echo "debian binary option set"
            DEBIAN_BUILD=1
            ;;
        k)
            echo "k8s binary option set"
            K8S_BUILD=1
            ;;
        \?)
            echo "Invalid option"
            exit 1 ;;
    esac
done

if [ -d "/device-config-manager" ]; then
    echo "Sucessfully mounted DCM directory!"
else
    echo "Mounting DCM directory unsuccessful!"
fi

cd /device-config-manager
TOP_DIR=$(pwd)
echo "Current directory: $(pwd)"
rm -rf $TOP_DIR/bin/device-config-manager-$UBUNTU_VERSION

mkdir -p $TOP_DIR/build/assets/
mkdir -p $TOP_DIR/bin

if [ "$UBUNTU_VERSION" = "jammy" ]; then
    cp -r $TOP_DIR/assets/amd_smi_lib/x86_64/$UBUNTU_LIBDIR/lib/* $TOP_DIR/build/assets
elif [ "$UBUNTU_VERSION" = "noble" ]; then
    cp -r $TOP_DIR/assets/amd_smi_lib/x86_64/$UBUNTU_LIBDIR/lib/* $TOP_DIR/build/assets
fi 

if [ "$K8S_BUILD" == 1 ]; then
    echo "Building DCM binary for $UBUNTU_VERSION"
    go build -ldflags "-s -w -X main.Version=$VERSION -X main.GitCommit=$GIT_COMMIT -X main.BuildDate=$BUILD_DATE " -o dcm_build $TOP_DIR/cmd/deviceconfigmanager/main.go

    if [ $? -ne 0 ]; then
    echo "DCM build failed. Exiting..."
    exit 1
    fi

    echo "Sucessfully build DCM binary for $UBUNTU_VERSION"
    cp dcm_build $TOP_DIR/bin/device-config-manager-$UBUNTU_VERSION
    rm -rf dcm_build
fi

rm -rf $TOP_DIR/build/
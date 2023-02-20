#!/bin/bash -e
error() {
  printf '\E[31m'; echo "$@"; printf '\E[0m'
}

# it's enough if current user is member of docker group
if [ `groups | grep -c docker` -eq 0 -a $EUID -ne 0 ]; then
    error "This script should be run by a user with docker access"
    exit 1
fi

export TAG=${1:-latest}
export ARCH=${2:-"$(uname -m)"}

# our local repository
export REPO="swarm-registry:5000"

build() {
    if [ $ARCH == "armv7l" ]
    then
        BPLATFORM="linux/arm/v7"
    else if [ $ARCH == "aarch64" ] 
        then
            BPLATFORM="linux/arm64"
        else
            BPLATFORM="linux/amd64"
        fi
    fi
    if [ $ARCH == "x86_64" ]
    then
        docker plugin rm -f $REPO/$1:$TAG || true
    else
        docker plugin rm -f $REPO/$1-$ARCH:$TAG || true
    fi
    docker rmi -f rootfsimage || true
    docker buildx build --load --platform ${BPLATFORM} \
        --build-arg GO_VERSION=1.15.10 \
        --build-arg UBUNTU_VERSION=20.04 \
        -t rootfsimage -f $1/Dockerfile .
    id=$(docker create rootfsimage true) # id was cd851ce43a403 when the image was created
    rm -rf build/rootfs
    mkdir -p build/rootfs/var/lib/docker-volumes
    docker export "$id" | tar -x -C build/rootfs
    docker rm -vf "$id"
    cp $1/config.json build
    if [ $ARCH == "x86_64" ]
    then
        docker plugin create $REPO/$1:$TAG build
        docker plugin push $REPO/$1:$TAG
	docker plugin rm -f $REPO/$1:$TAG || true        
    else
        docker plugin create $REPO/$1-$ARCH:$TAG build
        docker plugin push $REPO/$1-$ARCH:$TAG
        docker plugin rm -f $REPO/$1-$ARCH:$TAG || true        
    fi
}
build glusterfs-volume-plugin
#build s3fs-volume-plugin
#build cifs-volume-plugin

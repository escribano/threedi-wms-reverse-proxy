#!/bin/bash

# build script that injects version from ``version`` file into the ``main.Version`` variable, so that the version gets compiled into the binary
# also see: http://stackoverflow.com/questions/28459102/golang-compile-environment-variable-into-binary
export VERSION=`cat version`
go build -ldflags "-X main.Version=$VERSION" -o wmsrp .


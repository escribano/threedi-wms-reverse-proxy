#!/bin/bash

export VERSION=`cat version`
go build -ldflags "-X main.Version=$VERSION" -o wmsrp .


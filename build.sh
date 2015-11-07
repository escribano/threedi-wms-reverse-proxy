#!/bin/bash

export VERSION=`cat version.txt` 
go build -ldflags "-X main.Version=$VERSION" -o wmsrp .


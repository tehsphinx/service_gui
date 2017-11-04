#!/usr/bin/env bash

CURPATH=$(pwd)
cd ${GOPATH}/src/git.raceresult.com/LocalAdapterServer/gui

astilectron-bundler -v

go run *.go -v -d
#open ./output/darwin-amd64/LocalAdapter.app --args -v -d

cd ${CURPATH}

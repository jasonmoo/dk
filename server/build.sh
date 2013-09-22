#!/bin/bash

TARGET=${1?:"Please specify a build target: darwin/386, darwin/amd64, freebsd/386, freebsd/amd64, linux/386, linux/amd64"}
RECOMPILE=${2}

MY_BUILD_DATE=$(date | perl -pe "s/\n//g")
MY_BUILD_BUILDER=$(whoami)
MY_BUILD_SYSTEM=$(uname -a)
MY_BUILD_REPO=$(git remote show -n origin | perl -ne "print (/Fetch URL: (.+)$/);")
MY_BUILD_BRANCH=$(git rev-parse --abbrev-ref HEAD)
MY_BUILD_COMMIT=$(git rev-parse master)

cat << EOF > my_build_info.go
package main

import (
	"fmt"
	"runtime"
)

var BuildInfo = fmt.Sprintf(\`
       ._____
     __| _/  | __
    / __ ||  |/ /
   / /_/ ||    <
   \____ ||__|_ \\
        \/     \/

        v 0,1

Build Info
Date:    $MY_BUILD_DATE
Builder: $MY_BUILD_BUILDER
System:  $MY_BUILD_SYSTEM
Origin:  $MY_BUILD_REPO
Rev:     $MY_BUILD_BRANCH/$MY_BUILD_COMMIT

Go Build Info
Version: %s
GOROOT:  %s, OS: %s, Arch: %s, Compiler: %s
\`, runtime.Version(), runtime.GOROOT(), runtime.GOOS, runtime.GOARCH, runtime.Compiler)

func init() {
	fmt.Println(BuildInfo)
}

EOF

export GOROOT=/tmp/go
export GOBIN=$GOROOT/bin
export GOPATH=$(pwd)
export GOOS=${TARGET%/*}
export GOARCH=${TARGET#*/}

date
echo "dk build script starting up"
echo

if [[ -z "$RECOMPILE" ]]; then
	(
		echo "Ensuring $GOROOT"   && mkdir -p $GOROOT && cd $GOROOT &&
		echo "Downloading source" && ([ -d .hg ] || hg clone https://code.google.com/p/go .) && hg pull && hg up default &&
		echo "Building stdlib"    && cd src && ./make.bash --no-clean 2>&1
	)
	if [[ $? -ne 0 ]]; then
		echo "Go build failed.  Exiting..." && exit 1
	fi
fi


# grab deps, clear existing builds, and build
echo "go get..."   && $GOBIN/go get &&
echo "go build..." && $GOBIN/go build -o "bin/dk-$GOOS-$GOARCH" dk.go my_build_info.go
r=$?

# cleanup
rm -f my_build_info.go

if [[ $r -ne 0 ]]; then
	echo "dk build failed.  Exiting..." && exit 1
fi

# status
ls -lh bin
md5  bin/* | column -t
file bin/*

echo "Done!"
echo

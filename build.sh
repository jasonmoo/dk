#!/bin/bash

TARGET=${1?:"Please specify a build target: darwin/386, darwin/amd64, freebsd/386, freebsd/amd64, freebsd/arm, linux/386, linux/amd64, linux/arm, windows/386, windows/amd64"}
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
export GOPATH=`pwd`
export GOOS=${TARGET%/*}
export GOARCH=${TARGET#*/}

if [[ -n "$RECOMPILE" ]]; then
	(
		echo "Pulling new go source tarball"
		rm -rf $GOROOT && cd /tmp &&
		curl "https://go.googlecode.com/files/go1.1.2.src.tar.gz" | tar xz &&
		echo "Source retrieved.  Building stdlib" &&
		cd go/src && ./make.bash --no-clean 2>&1
	)
fi


# grab deps, clear existing builds, and build
echo "Building..."
(rm -f bin/* && $GOBIN/go get && $GOBIN/go build -o "bin/dk-$GOOS-$GOARCH" dk.go my_build_info.go)
r=$?

# cleanup
rm -f my_build_info.go

# status
ls -lh bin
md5  bin/* | column -t
file bin/*

echo "Done!"

exit $r

#!/bin/bash

set -e
set -x

REV=$(git rev-parse --short HEAD)
TAG=$(git describe --tags)
VERSION="${TAG} (${REV})"
GO_OSARCH="darwin/amd64 linux/386 linux/amd64 linux/arm windows/386 windows/amd64"

mkdir -p dist

for v in ${GO_OSARCH}; do
	GOOS=$(echo ${v} | cut -d/ -f 1)
	GOARCH=$(echo ${v} | cut -d/ -f 2)
	GOOS=${GOOS} GOARCH=${GOARCH} go build -ldflags "-X \"main.appVersion=${VERSION}\""
	exe=max7456tool
	rel=max7456tool_${TAG}_${GOOS}_${GOARCH}
	zip=${rel}.zip
	tgz=${rel}.tar.gz
	if [ ${GOOS} = "windows" ]; then
	    exe=${exe}.exe
	    zip ${zip} ${exe}
            dist=${zip}
        elif [ ${GOOS} = "darwin" ]; then
            if ! [ -z ${CODESIGN} ]; then
                macapptool sign ${exe}
                macapptool notarize -u "${NOTARIZATION_USERNAME}" -p "${NOTARIZATION_PASSWORD}" ${exe}
            fi
            zip ${zip} ${exe}
            dist=${zip}
        else
		tar czf ${tgz} ${exe}
                dist=${tgz}
	fi
        mv ${dist} dist
done

go clean

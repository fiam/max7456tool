#!/bin/bash
VERSION=$(grep app.Version main.go | cut -d= -f 2 | tr -d '[:space:]"')
GO_OSARCH="darwin/amd64 linux/386 linux/amd64 linux/arm windows/386 windows/amd64"

mkdir -p dist

for v in ${GO_OSARCH}; do
	GOOS=$(echo ${v} | cut -d/ -f 1)
	GOARCH=$(echo ${v} | cut -d/ -f 2)
	GOOS=${GOOS} GOARCH=${GOARCH} go build
	exe=max7456tool
	rel=max7456tool_${VERSION}_${GOOS}_${GOARCH}
	if [ ${GOOS} = "windows" ]; then
		exe=${exe}.exe
		zip=${rel}.zip
		zip ${zip} ${exe}
		mv ${zip} dist
	else
		tgz=${rel}.tar.gz
		tar czf ${tgz} ${exe}
		mv ${tgz} dist
	fi
done

go clean
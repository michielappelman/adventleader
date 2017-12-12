all: local

STAMP   := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
GITHASH := $(shell git rev-parse HEAD)
LDFLAGS  = -ldflags "-X main.buildstamp=${STAMP} -X main.githash=${GITHASH}"

linux:
	env GOOS=linux GOARCH=amd64 go build ${LDFLAGS}

local:
	go build ${LDFLAGS}

clean:
	rm adventleader

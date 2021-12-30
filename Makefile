VERSION := $(shell cat VERSION)
GIT_COMMIT := $(shell git rev-parse HEAD)
BRANCH := $(shell git rev-parse --abbrev-ref HEAD)
PROJECT := netcupscp-exporter
PKG := github.com/prometheus/common

.PHONY: build
build:
	go build -ldflags "-s -w -X ${PKG}/version.Version=${VERSION} -X ${PKG}/version.Revision=${GIT_COMMIT} -X ${PKG}/version.Branch=${BRANCH} -X ${PKG}/version.BuildUser=${USER}@${HOST} -X ${PKG}/version.BuildDate=${BUILD_DATE}" -o ${PROJECT} .

.PHONY: lint
lint:
	golangci-lint run ./...

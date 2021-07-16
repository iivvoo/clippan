default: clippan

BUILD_TIME=$(shell date +%FT%T%z)
GIT_REVISION=$(shell git rev-parse --short HEAD)
GIT_BRANCH=$(shell git rev-parse --symbolic-full-name --abbrev-ref HEAD)
GIT_DIRTY=$(shell git diff-index --quiet HEAD -- || echo "x-")

LDFLAGS=-ldflags "-s -X main.BuildStamp=$(BUILD_TIME) -X main.GitHash=$(GIT_DIRTY)$(GIT_REVISION) -X main.gitBranch=$(GIT_BRANCH)"


srcfiles = cmd/clippan/main.go */*.go

testpackages = ./...

default: bin/clippan

bin/clippan: $(srcfiles)
	go build -o bin/clippan $(LDFLAGS) cmd/clippan/main.go

lint:
	golangci-lint run

test-dev: lint test

test: 
	go test -count=1 $(testpackages)

test-verbose:
	go test -v $(testpackages)

run-dev:
	CompileDaemon -command=bin/clippan -build="make"

mod-tidy:
	go mod tidy

show-deps:
	go list -m all

clean:
	@rm -f bin/*


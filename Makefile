
.PHONY: build
build: build_darwin

OUT_FILE=tup
MAIN_PACKAGE:=`go list -f "{{.Name}}|{{.ImportPath}}" ../...|grep "main|" | sed s:"main|":"":g`
PACKAGES:=$(shell go list ./... 2>/dev/null)

GIT_SHA:=$(shell git log -n 1 --pretty=format:'%H')
BUILD_FLAGS:='-X main.GitSha=${GIT_SHA}'

.PHONY: test
test:
	go test -race -v $(PACKAGES)

.PHONY: build_darwin
build_darwin:
	export GOOS=$(subst build_,,$@) && export GOARCH="amd64" && mkdir -p build/$${GOOS}_$${GOARCH} && cd build &&\
		go build --ldflags ${BUILD_FLAGS} -o ${OUT_FILE}_$${GOOS}_$${GOARCH} ${MAIN_PACKAGE} && cp ${OUT_FILE}_$${GOOS}_$${GOARCH} $${GOOS}_$${GOARCH}/${OUT_FILE}

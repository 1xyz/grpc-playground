GO=go
GOFMT=gofmt
PROTOC=protoc
DELETE=rm
BINARY=grpc-play
BUILD_BINARY=bin/$(BINARY)
# go source files, ignore vendor directory
SRC = $(shell find . -type f -name '*.go' -not -path "./vendor/*")
# current git version short-hash
VER = $(shell git rev-parse --short HEAD)
TAG = "$(BINARY):$(VER)"

info:
	@echo " target         ⾖ Description.                                    "
	@echo " ----------------------------------------------------------------- "
	@echo
	@echo " build          generate a local build ⇨ $(BUILD_BINARY)          "
	@echo " clean          clean up bin/ & go test cache                      "
	@echo " fmt            format go code files using go fmt                  "
	@echo " protoc         compile proto files to generate go files           "
	@echo " ------------------------------------------------------------------"

build: clean fmt protoc
	$(GO) build -o $(BUILD_BINARY) -v main.go


.PHONY: clean
clean:
	$(DELETE) -rf bin/
	$(GO) clean -cache


.PHONY: fmt
fmt:
	$(GOFMT) -l -w $(SRC)


.PHONY: protoc
protoc:
	$(GO) get -u github.com/golang/protobuf/protoc-gen-go
	$(PROTOC) -I api/ api/*.proto --go_out=plugins=grpc:api/ --go_opt=paths=source_relative

.PHONY: server
server: protoc
	$(GO) build -o bin/server -v server/main.go

.PHONY: client
client: protoc
	$(GO) build -o bin/client -v client/main.go
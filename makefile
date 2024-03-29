PROTOFILES = $(shell find ./ -type f -name '*.proto')
PBGOFILES = $(patsubst %.proto, %.pb.go, $(PROTOFILES))
GOFILES = $(shell find ./ -type f -name '*.go')
TAGS ?= azurite
FINAL_TAGS = $(TAGS) all

bart: $(GOFILES) $(PBGOFILES)
	go build -tags "$(FINAL_TAGS)"

.PHONY: build
build: linux

.PHONY: all
all: linux windows

.PHONY: my
my: bart-linux-amd64

.PHONE: tags
tags:
	go build -o _out/bart-azure -tags "azure all"
	go build -o _out/bart-azurite -tags "azurite all"
	go build -o _out/bart-files -tags "files all"

test-azurite-backup:
	go build -tags "azurite all"
	./bart -logtostderr=true -v=2 backup -path _out/ -name test-bart 

test-azurite-restore:
	go build -tags "azurite all"
	./bart -logtostderr=true -v=2 restore -path _out/ -name test-bart

test-files-backup:
	go build -tags "files all"
	./bart -logtostderr=true -v=2 backup -path _out/ -name test-bart

test-files-restore:
	go build -tags "files all"
	./bart -logtostderr=true -v=2 restore -path _out/ -name test-bart

linux: bart-linux-386 bart-linux-amd64 bart-linux-arm bart-linux-arm64
windows: bart-windows-386.exe bart-windows-amd64.exe

bart-windows-386.exe: $(GOFILES) $(PBGOFILES)
	GOOS=windows GOARCH=386 go build -o _out/bart-windows-386.exe -tags "$(FINAL_TAGS)"

bart-windows-amd64.exe: $(GOFILES) $(PBGOFILES)
	GOOS=windows GOARCH=amd64 go build -o _out/bart-windows-amd64.exe -tags "$(FINAL_TAGS)"

bart-linux-386: $(GOFILES) $(PBGOFILES)
	GOOS=linux GOARCH=386 go build -o _out/bart-linux-386 -tags "$(FINAL_TAGS)"

bart-linux-amd64: $(GOFILES) $(PBGOFILES)
	GOOS=linux GOARCH=amd64 go build -o _out/bart-linux-amd64 -tags "$(FINAL_TAGS)"

bart-linux-arm: $(GOFILES) $(PBGOFILES)
	GOOS=linux GOARCH=arm go build -o _out/bart-linux-arm -tags "$(FINAL_TAGS)"

bart-linux-arm64: $(GOFILES) $(PBGOFILES)
	GOOS=linux GOARCH=arm64 go build -o _out/bart-linux-arm64 -tags "$(FINAL_TAGS)"

%.pb.go: %.proto
	@go generate ./...

update-dependencies:
	@go get -u ./...
	@go mod tidy

clean:
	@rm -f bart bart.exe || true
	@find ./ -type f -name '*.pb.go' -delete || true

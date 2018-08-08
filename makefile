PROTOFILES = $(shell find ./ -type f -name '*.proto')
PBGOFILES = $(patsubst %.proto, %.pb.go, $(PROTOFILES))
GOFILES = $(shell find ./ -type f -name '*.go')

build: bart

all: bart windows

windows: bart.x86.exe bart.x64.exe

bart.x86.exe: $(GOFILES) $(PBGOFILES)
	GOOS=windows GOARCH=386 go build -o bart.x86.exe

bart.x64.exe: $(GOFILES) $(PBGOFILES)
	GOOS=windows GOARCH=amd64 go build -o bart.x64.exe

bart: $(GOFILES) $(PBGOFILES)
	go build -o bart

%.pb.go: %.proto
	protoc --go_out=. $<

clean:
	rm -f bart bart.x86.exe bart.x64.exe || true
	find ./ -type f -name '*.pb.go' -delete || true

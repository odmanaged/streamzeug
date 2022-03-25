SOURCES != find . -name '*.go'
ROOT_DIR:=$(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))
DTAPI_INCLUDE=$(ROOT_DIR)/dektec/DTAPI/Include


all: binaries

binaries: streamzeug

dektecasi: dektec/asi.cpp dektec/asi.h
	CPATH=$(DTAPI_INCLUDE) g++ -c dektec/asi.cpp -o dektecasi.o

libdektec: dektecasi
	ar rcs ./libdektec.a dektecasi.o dektec/DTAPI/Lib/GCC5.1_CXX11_ABI1/DTAPI64.o

streamzeug: bin/streamzeug
bin/streamzeug: $(SOURCES) go.mod go.sum libdektec
	LIBRARY_PATH=$(ROOT_DIR) go build -o bin/streamzeug ./cmd/streamzeug

.PHONY: install
install: streamzeug
	install -o 0 -g 0 bin/streamzeug /usr/bin/streamzeug

.PHONY:
test:
	go test -v ./...

.PHONY:
lint:
	golangci-lint run

.PHONY:
clean:
	rm -rf bin libdektec.a dektecasi.*
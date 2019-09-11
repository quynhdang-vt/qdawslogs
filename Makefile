
BINARY := qdawslogs
OS := $(shell uname -s)
ifeq ($(OS), Darwin)
	LOC:=osx
else
	LOC:=linux
endif

.PHONY: build
build:
	go build -o $(BINARY)
	mv $(BINARY) $(LOC)

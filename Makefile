
.PHONY: build-mac
build-mac:
	go build -o qdawslogs.mac

.PHONY: build
build:
	go build -o qdawslogs

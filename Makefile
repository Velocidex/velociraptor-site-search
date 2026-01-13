all:
	go build -o ./search ./cmd/

test:
	go test -v ./...

build:
	rm -rf /tmp/index/
	go run ./cmd/ build ../velociraptor-docs/content/ /tmp/index/
	cd /tmp/index && zip -r ../index.zip *

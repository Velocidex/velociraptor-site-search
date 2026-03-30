all:
	go build -o ./search ./cmd/

test:
	go test -v ./...

serve:
	go run ./cmd/ serve server.config.yaml

debug:
	dlv debug ./cmd/ -- serve server.config.yaml

build:
	rm -rf /tmp/index/
	go run ./cmd/ build ../velociraptor-docs/content/ /tmp/index/
	cd /tmp/index && zip -r ../index.zip *

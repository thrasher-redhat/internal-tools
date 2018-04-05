TAG?=dev

%.go:

bin/%: cmd/%/*.go pkg/**/*.go
	CGO_ENABLED=0 go build -o ./bin/$* ./cmd/$*/...

all: bin/snapshot bin/serve
images: snapshot-image serve-image
.PHONY: all images %-image

%-image: bin/%
	cp ./bin/$* ./deploy/$*
	docker build -t $*:$(TAG) -f deploy/Dockerfile.$* deploy/
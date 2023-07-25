MODULE := github.com/jfk9w/hoarder

build:
	go build -v ./...

test:
	go test -v ./...

schema:
	go run main.go --dump.schema > config.schema.json

defaults:
	go run main.go --dump.values > config.defaults.json

config: schema defaults

tools:
	go install golang.org/x/tools/cmd/goimports@latest

fmt: tools
	goimports -local $(MODULE) -l -w $(shell find . -type f -name '*.go' -not -path "./vendor/*")

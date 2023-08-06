MODULE := github.com/jfk9w/hoarder
MAIN := cmd/hoarder/main.go

build:
	go build -v ./...

test:
	go test -v ./...

schema:
	mkdir -p config && go run $(MAIN) --dump.schema > config/schema.yaml

defaults:
	mkdir -p config && go run $(MAIN) --dump.values > config/defaults.json

config: schema defaults

tools:
	go install golang.org/x/tools/cmd/goimports@latest

fmt: tools
	goimports -local $(MODULE) -l -w $(shell find . -type f -name '*.go' -not -path "./vendor/*")

MODULE := $(shell head -1 go.mod | cut -d ' ' -f2)
NAME := $(shell head -1 go.mod | cut -d '/' -f3)
GOIMPORTS := $(shell go env GOPATH)/bin/goimports

export GOEXPERIMENT := loopvar

$(GOIMPORTS):
	go install golang.org/x/tools/cmd/goimports@latest

fmt: $(GOIMPORTS)
	$(GOIMPORTS) -local $(MODULE) -l -w $(shell find . -type f -name '*.go' -not -path "./vendor/*")

test:
	go test -v ./...

bin/%:
	go build -o $@ -v ./$(subst bin,cmd,$@)

bin: $(subst cmd,bin,$(wildcard ./cmd/*))

config/schema.yaml: bin/hoarder
	mkdir -p $(shell dirname $@) && ./$^ --dump.schema > config/schema.yaml

config/defaults.json: bin/hoarder
	mkdir -p $(shell dirname $@) && ./$^ --dump.values > config/defaults.json

config: config/schema.yaml config/defaults.json

install: bin
	cp bin/* /usr/local/bin/

uninstall:
	rm $(subst ./cmd,/usr/local/bin,$(wildcard ./cmd/*))

clean:
	rm -rf bin/*

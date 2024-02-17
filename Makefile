MODULE := $(shell head -1 go.mod | cut -d ' ' -f2)

GOIMPORTS := $(shell go env GOPATH)/bin/goimports
OGEN := $(shell go env GOPATH)/bin/ogen

export GOEXPERIMENT := loopvar

$(GOIMPORTS):
	go install golang.org/x/tools/cmd/goimports@latest

$(OGEN):
	go install github.com/ogen-go/ogen/cmd/ogen@v0.76.0

fmt: $(GOIMPORTS)
	$(GOIMPORTS) -local $(MODULE) -l -w $(shell find . -type f -name '*.go' -not -path "./vendor/*")

gen: $(OGEN)
	go generate ./...

test: gen
	go test -v ./...

bin/%: gen $(wildcard ./internal/**/*) $(wildcard ./cmd$@/**/*)
	go build -o $@ -v ./$(subst bin,cmd,$@)

bin: $(subst ./cmd,bin,$(wildcard ./cmd/*))

%/schema.yaml: bin/hoarder
	mkdir -p $(dir $@) && ./$^ --dump.schema > $@

%/defaults.json: bin/hoarder
	mkdir -p $(dir $@) && ./$^ --dump.values > $@

config: config/schema.yaml config/defaults.json

install: bin
	cp bin/* /usr/local/bin/

uninstall:
	rm -f $(subst ./cmd,/usr/local/bin,$(wildcard ./cmd/*))

clean:
	rm -rf bin/*

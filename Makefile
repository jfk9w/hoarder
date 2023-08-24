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

bin/%: $(wildcard ./internal/**/*) $(wildcard ./cmd$@/**/*)
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

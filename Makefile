MODULE := $(shell head -1 go.mod | cut -d ' ' -f2)
CMD := ./cmd

ifndef BUILD
BUILD := bin
endif

export GOEXPERIMENT := loopvar

$(BUILD)/%:
	mkdir -p $(BUILD) && go build -o $@ -v $(subst $(BUILD),$(CMD),$@)

build: $(subst $(CMD),$(BUILD),$(wildcard $(CMD)/*))
	echo $^

test:
	go test -v ./...

schema: $(BUILD)/hoarder
	mkdir -p config && ./$^ --dump.schema > config/schema.yaml

defaults: $(BUILD)/hoarder
	mkdir -p config && ./$^ --dump.values > config/defaults.json

config: schema defaults

tools:
	go install golang.org/x/tools/cmd/goimports@latest

fmt: tools
	goimports -local $(MODULE) -l -w $(shell find . -type f -name '*.go' -not -path "./vendor/*")

clean:
	rm -rf $(BUILD)/*

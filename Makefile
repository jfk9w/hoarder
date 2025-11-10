MODULE := $(shell head -1 go.mod | cut -d ' ' -f2)

gen: 
	go generate ./...

test: gen
	go test -v ./...

bin/%: gen $(wildcard ./internal/**/*) $(wildcard ./cmd$@/**/*)
	go build -o $@ -v ./$(subst bin,cmd,$@)

bin: $(subst ./cmd,bin,$(wildcard ./cmd/*))

%/schema.json: bin/hoarder
	mkdir -p $(dir $@) && ./$^ --dump.schema > $@

%/defaults.json: bin/hoarder
	mkdir -p $(dir $@) && ./$^ --dump.values > $@

config: config/schema.json config/defaults.json

install: bin
	cp bin/* /usr/local/bin/

uninstall:
	rm -f $(subst ./cmd,/usr/local/bin,$(wildcard ./cmd/*))

clean:
	rm -rf bin/*

OUTFILES := $(patsubst cmd/%.go,bin/%,$(wildcard cmd/*.go))

.mod:
	go mod download
	touch .mod

bin/%: cmd/%.go
	go build -o $@ $<

all: $(OUTFILES) .mod

clean:
	rm -rf $(OUTFILES) bin .mod
	go mod tidy

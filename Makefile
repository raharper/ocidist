BINS := bin/ocidist

.PHONY: all clean
all: $(BINS)

clean:
	rm -f -v $(BINS)

bin/ocidist: cmd/ocidist/*.go cmd/ocidist/cmd/*.go pkg/*/*.go
	go build -o $@ cmd/ocidist/*.go

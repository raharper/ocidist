BINS := bin/ocidist
TAGS := containers_image_openpgp -s -w
TESTS := $(shell find tests -type f -name *_test.go)

.PHONY: all
all: test $(BINS)

.PHONY: go-download
go-download:
	go mod download

.PHONY: test
test: $(TESTS)
	go test $<

.PHONY: clean
clean:
	rm -f -v $(BINS)

bin/ocidist: cmd/ocidist/*.go cmd/ocidist/cmd/*.go pkg/*/*.go
	go build -tags "$(TAGS)" -o $@ cmd/ocidist/*.go

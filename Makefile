BINS := bin/ocidist
TAGS := containers_image_openpgp -s -w

.PHONY: all clean
all: $(BINS)

clean:
	rm -f -v $(BINS)

bin/ocidist: cmd/ocidist/*.go cmd/ocidist/cmd/*.go pkg/*/*.go
	go build -tags "$(TAGS)" -o $@ cmd/ocidist/*.go

update-api-docs:
	(cd pkg/api; gomarkdoc -o README.md)

.PHONY: build run test clean

BINARY := external-link-tracker
BUILDFILES := main.go handlers.go

build: $(BINARY)

run: _vendor
	gom run $(BUILDFILES)

test: _vendor
	LINK_TRACKER_MONGO_DB=external_link_tracker_test gom test

clean:
	rm -f $(BINARY)

_vendor: Gomfile
	gom install
	touch _vendor

$(BINARY): _vendor $(BUILDFILES)
	gom build -o $(BINARY) $(BUILDFILES)

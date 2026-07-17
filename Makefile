# go-edit Makefile — mirrors sibling repos (go-term, go-kite).
# `make app` packages examples/npad as a macOS .app bundle ready
# to drop into /Applications.

.PHONY: test test-race vet lint build app clean-app clean

DEMO_BIN     := npad
APP_NAME     := Npad
BUILDAPP_DIR := ../go-gui/cmd/buildapp
BUILDAPP_BIN := $(BUILDAPP_DIR)/buildapp

test:
	go test ./edit/...

test-race:
	go test -race ./edit/...

vet:
	go vet ./...

lint:
	golangci-lint run

build:
	go build ./...

# Package npad as a macOS .app bundle.
app: $(APP_NAME).app

$(BUILDAPP_BIN):
	cd $(BUILDAPP_DIR) && go build -o buildapp .

$(APP_NAME).app: $(BUILDAPP_BIN)
	cd examples/npad && go build -o $(CURDIR)/$(DEMO_BIN) .
	$(BUILDAPP_BIN) -bundle-deps -o . -name $(APP_NAME) \
		-id github.com.go-gui-org.go-edit $(DEMO_BIN)

clean-app:
	rm -f $(DEMO_BIN)
	rm -rf $(APP_NAME).app
	cd $(BUILDAPP_DIR) && rm -f buildapp

# Clean test cache and built binaries.
clean:
	go clean -testcache

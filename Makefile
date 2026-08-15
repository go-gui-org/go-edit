# go-edit Makefile — mirrors sibling repos (go-term, go-kite).
# `make app` packages examples/npad as a macOS .app bundle ready
# to drop into /Applications.

.PHONY: test test-race vet lint build build-examples prepush app clean-app clean

DEMO_BIN     := npad
APP_NAME     := Npad
BUILDAPP_DIR := ../go-gui/cmd/buildapp
BUILDAPP_BIN := $(BUILDAPP_DIR)/buildapp

# Gate recipes resolve modules from go.mod, not from a go.work workspace.
# CI never sees a workspace file, so a gate that used one would answer a
# different question than "will CI go green". The app build targets below
# deliberately keep a bare `go` so local development against a sibling
# go-gui checkout still works.
GO := GOWORK=off go

# golangci-lint is its own binary, so $(GO) does not cover it — but it
# honours go.work the same way the toolchain does. Without GOWORK=off it
# would type-check against sibling working copies and report breakage that
# CI, which builds the pinned versions, will never see.
LINT := GOWORK=off golangci-lint

# CI scopes tests to ./edit/... — examples are built, not tested.
test:
	$(GO) test ./edit/...

# Race-enabled tests. CI runs -race on its Linux runner only; running it
# here covers that leg from any host. -shuffle=on and -count=1 come from
# the old scripts/ci.sh, which this target replaces: shuffling catches
# order-dependent tests that a fixed order hides.
test-race:
	$(GO) test -race -count=1 -shuffle=on ./edit/...

# Static analysis. Broader than CI's ./edit/... — this also covers the
# example programs, which CI only ever compiles.
vet:
	$(GO) vet ./...

lint:
	$(LINT) run

build:
	$(GO) build ./...

# Compile the example programs. Always pass an explicit -o under build/:
# a bare `go build ./examples/basic` drops a binary in the repo root.
build-examples:
	$(GO) build -o build/basic ./examples/basic
	$(GO) build -o build/npad ./examples/npad

# Recommended full local validation before pushing (issue go-gui#314).
# Approximates the CI matrix from one host: race tests, vet, lint, and the
# example builds. Aborts on the first failing target.
#
# Omissions vs CI, by design: the OS matrix itself — CI runs the suite on
# both ubuntu-latest and macos-latest, and only the host's own platform is
# exercised here.
prepush: test-race vet lint build-examples

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

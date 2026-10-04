# go-edit Makefile — mirrors sibling repos (go-term, go-kite).
# `make app` packages examples/npad as a macOS .app bundle ready
# to drop into /Applications.

.PHONY: test test-race vet lint lint-bin build build-examples prepush app clean-app clean

DEMO_BIN     := npad
APP_NAME     := Npad
BUILDAPP_DIR := ../go-gui/cmd/buildapp
BUILDAPP_BIN := $(BUILDAPP_DIR)/buildapp
# Code-signing identity for the bundle. Empty (the default) means buildapp
# signs ad-hoc, and an ad-hoc signature has no certificate for TCC to key a
# permission grant against — TCC falls back to the cdhash, which changes on
# every build, so each `make app` silently revokes any granted permission
# while System Settings keeps showing it as granted. Set this to a
# self-signed code-signing certificate to keep grants across rebuilds:
#   make app SIGN_IDENTITY="My Dev Cert"
# BUILDAPP_SIGN_IDENTITY in the environment does the same without the flag.
SIGN_IDENTITY ?=
SIGN_FLAG    := $(if $(SIGN_IDENTITY),-sign "$(SIGN_IDENTITY)",)

# Gate recipes resolve modules from go.mod, not from a go.work workspace.
# CI never sees a workspace file, so a gate that used one would answer a
# different question than "will CI go green". The app build targets below
# deliberately keep a bare `go` so local development against a sibling
# go-gui checkout still works.
GO := GOWORK=off go

# Repo-local bin for the pinned linter. The pinned VERSION itself lives in
# tools/lint/go.mod -- see the $(LINT_BIN) rule below. `make lint` and CI
# both build from that file, so a local pass and a CI pass run one version.
LINT_DIR = $(CURDIR)/.bin
LINT_BIN = $(LINT_DIR)/golangci-lint

# golangci-lint is its own binary, so $(GO) does not cover it — but it
# honours go.work the same way the toolchain does. Without GOWORK=off it
# would type-check against sibling working copies and report breakage that
# CI, which builds the pinned versions, will never see.
LINT := GOWORK=off $(LINT_BIN)

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

# Build the pinned golangci-lint into .bin/. It rebuilds only when
# tools/lint/go.mod or go.sum change. GOWORK=off keeps a local go.work out
# of the build. GOOS/GOARCH/CGO_ENABLED are cleared so a caller that sets
# them to pick a lint target does not cross-compile the linter itself into
# a binary this host cannot run.
$(LINT_BIN): tools/lint/go.mod tools/lint/go.sum
	GOWORK=off GOOS= GOARCH= CGO_ENABLED=0 GOFLAGS= GOBIN=$(LINT_DIR) \
	  go -C tools/lint install \
	  github.com/golangci/golangci-lint/v2/cmd/golangci-lint

lint-bin: $(LINT_BIN)

lint: $(LINT_BIN)
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
		-id github.com.go-gui-org.go-edit $(SIGN_FLAG) $(DEMO_BIN)

clean-app:
	rm -f $(DEMO_BIN)
	rm -rf $(APP_NAME).app
	cd $(BUILDAPP_DIR) && rm -f buildapp

# Clean test cache and built binaries.
clean:
	go clean -testcache

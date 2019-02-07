# Set the mode for code-coverage
GO_TEST_COVERAGE_MODE ?= count
GO_TEST_COVERAGE_FILE_NAME ?= coverage.out

# Set a default `min_confidence` value for `golint`
GO_LINT_MIN_CONFIDENCE ?= 0.2

all: test

.PHONY: test
test:
	@echo "Run unit tests"
	@go test -v ./...

.PHONY: test-with-coverage
test-with-coverage:
	@echo "Run unit tests with coverage"
	@go test -cover ./...

.PHONY: test-with-coverage-profile
test-with-coverage-profile:
	@echo "Run unit tests with coverage profile"
	@echo "mode: ${GO_TEST_COVERAGE_MODE}" > "${GO_TEST_COVERAGE_FILE_NAME}"
	@go test -coverpkg=`go list ./... | grep -vE 'mock' | tr '\n' ','` -covermode ${GO_TEST_COVERAGE_MODE} -coverprofile=${GO_TEST_COVERAGE_FILE_NAME} ./...
	@echo "Generate coverage report";
	@go tool cover -func="${GO_TEST_COVERAGE_FILE_NAME}";
	@rm "${GO_TEST_COVERAGE_FILE_NAME}";

.PHONY: fix
fix: fix-format fix-import

.PHONY: fix-import
fix-import:
	@echo "Fix imports"
	@errors=$$(goimports -l -w -local $(GO_PKG) $$(go list -f "{{ .Dir }}" ./...)); if [ "$${errors}" != "" ]; then echo "$${errors}"; fi

.PHONY: fix-format
fix-format:
	@echo "Fix formatting"
	@gofmt -w ${GO_FMT_FLAGS} $$(go list -f "{{ .Dir }}" ./...); if [ "$${errors}" != "" ]; then echo "$${errors}"; fi

.PHONY: lint
lint: lint-format lint-import lint-style

.PHONY: lint-format
lint-format:
	@echo "Check formatting"
	@errors=$$(gofmt -l $$(go list -f "{{ .Dir }}" ./...)); if [ "$${errors}" != "" ]; then echo "Invalid format:\n$${errors}"; exit 1; fi

.PHONY: lint-import
lint-import:
	@echo "Check imports"
	@errors=$$(goimports -l $$(go list -f "{{ .Dir }}" ./...)); if [ "$${errors}" != "" ]; then echo "Invalid imports:\n$${errors}"; exit 1; fi

.PHONY: lint-style
lint-style:
	@echo "Check code style"
	@errors=$$(golint -min_confidence=${GO_LINT_MIN_CONFIDENCE} $$(go list ./...)); if [ "$${errors}" != "" ]; then echo "Invalid code style:\n$${errors}"; exit 1; fi

.PHONY: clean
clean:
	@echo "Cleanup"
	@find . -type f -name "*coverage*.out" -delete

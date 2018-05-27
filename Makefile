.PHONY: all test test-with-coverage test-with-coverage-profile lint lint-format lint-import lint-style clean

# Set the mode for code-coverage
GO_TEST_COVERAGE_MODE ?= count
GO_TEST_COVERAGE_FILE_NAME ?= coverage.out

# Set a default `min_confidence` value for `golint`
GO_LINT_MIN_CONFIDENCE ?= 0.2

all: test

test:
	@echo "Run unit tests"
	@go test -v ./...

test-with-coverage:
	@echo "Run unit tests with coverage"
	@go test -cover ./...

test-with-coverage-profile:
	@echo "Run unit tests with coverage profile"
	@echo "mode: ${GO_TEST_COVERAGE_MODE}" > "${GO_TEST_COVERAGE_FILE_NAME}"
	@go test -coverpkg=`go list ./... | grep -vE 'mock' | tr '\n' ','` -covermode ${GO_TEST_COVERAGE_MODE} -coverprofile=${GO_TEST_COVERAGE_FILE_NAME} ./...
	@echo "Generate coverage report";
	@go tool cover -func="${GO_TEST_COVERAGE_FILE_NAME}";
	@rm "${GO_TEST_COVERAGE_FILE_NAME}";

lint: lint-format lint-import lint-style

lint-format:
	@echo "Check formatting"
	@errors=$$(gofmt -l $$(go list -f "{{ .Dir }}" ./...)); if [ "$${errors}" != "" ]; then echo "Invalid format:\n$${errors}"; exit 1; fi

lint-import:
	@echo "Check imports"
	@errors=$$(goimports -l $$(go list -f "{{ .Dir }}" ./...)); if [ "$${errors}" != "" ]; then echo "Invalid imports:\n$${errors}"; exit 1; fi

lint-style:
	@echo "Check code style"
	@errors=$$(golint -min_confidence=${GO_LINT_MIN_CONFIDENCE} $$(go list ./...)); if [ "$${errors}" != "" ]; then echo "Invalid code style:\n$${errors}"; exit 1; fi

clean:
	@echo "Cleanup"
	@find . -type f -name "*coverage*.out" -delete

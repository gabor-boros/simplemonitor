.DEFAULT_GOAL := build

GO_EXEC := $(shell which go)

define log
	@echo "[\033[36mINFO\033[0m]\t$(1)" 1>&2;
endef

.PHONY: help
help: ## Show help message
	@echo "Available targets:";
	@grep -E '^[a-z.A-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}';

.PHONY: changelog
changelog: ## Update the changelog
	$(call log, updating changelog)

	echo "$(RELEASE_VERSION)" > /tmp/abba

	@if [ -z "$(RELEASE_VERSION)" ]; then \
		git cliff > CHANGELOG.md; \
	else \
		git cliff --tag "$(RELEASE_VERSION)" --unreleased > CHANGELOG.md; \
		git add CHANGELOG.md; \
		git commit -m "chore(changelog): update changelog for $(RELEASE_VERSION)"; \
	fi

.PHONY: release
release: ## Cut a new release
	$(call log, releasing new version)
	@goreleaser release --clean

.PHONY: dep
dep: ## Download dependencies
	$(call log, download backend dependencies)
	@$(GO_EXEC) mod tidy
	@$(GO_EXEC) mod download

.PHONY: build
build: ## Build
	$(call log, build monitor)
	@goreleaser build --single-target --snapshot --clean

.PHONY: lint
lint: ## Run linters
	$(call log, run backend linters)
	@golangci-lint run --timeout 5m

.PHONY: format
format: ## Run formatters
	$(call log, run formatters)
	@gofmt -l -s -w $(shell pwd)
	@goimports -w $(shell pwd)

.PHONY: clean
clean: destroy.backend ## Destroys all backend resources and cleans up untracked files
	$(call log, removing untracked files)
	@git clean -xd --force

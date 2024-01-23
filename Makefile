BREWPATH=/opt/homebrew
GOCMD=go
GOFUMPT=gofumpt
GOFUMPTPATH := $(GOPATH)/bin/gofumpt
GOFUMPTBREWPATH := $(BREWPATH)/bin/gofumpt

all: help

.PHONY: format
format:
	test -s ${GOFUMPTBREWPATH} || test -s ${GOFUMPTPATH} || $(GOCMD) install mvdan.cc/gofumpt@latest
	$(GOFUMPT) -l -w .

.PHONY: deploy
deploy:
	pulumi up

.PHONY: help
help:
	@echo ''
	@echo 'Usage:'
	@echo '  ${YELLOW}make${RESET} ${GREEN}<arg>${RESET}'
	@echo ''
	@echo 'Arguments:'
	@echo "  ${YELLOW}help       ${RESET} ${GREEN}Show this help message${RESET}"
	@echo "  ${YELLOW}format     ${RESET} ${GREEN}Format '*.go' files with gofumpt${RESET}"
	@echo "  ${YELLOW}deploy     ${RESET} ${GREEN}Deploy lambda function to AWS${RESET}"

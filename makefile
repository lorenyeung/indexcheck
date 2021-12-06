GOOS=linux
GOARCH=amd64
VERSION := $(shell jq -r '.script_version' metadata.json)
BINARY := $(shell jq -r '.binary_prefix' metadata.json)
PROJECT=github.com/lorenyeung/go-orchestrate
.PHONY: build

GIT_COMMIT := $(shell git rev-list -1 HEAD)

build:
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o $(BINARY)-linux-x64 -ldflags "-X $(PROJECT)/helpers.GitCommit=$(GIT_COMMIT) -X $(PROJECT)/helpers.Version=$(VERSION)" main.go
	GOOS=darwin GOARCH=$(GOARCH) go build -o $(BINARY)-darwin-x64 -ldflags "-X $(PROJECT)/helpers.GitCommit=$(GIT_COMMIT) -X $(PROJECT)/helpers.Version=$(VERSION)" main.go

clean:
	rm orchestrate-darwin-x64 orchestrate-linux-x64 

fmt-check:			## Gofmt check for errors
	./scripts/gofmt.sh

fmt-fix:			## Gofmt fix errors
	gofmt -w -s .

vet:				## GoVet xray
vet: fmt-check
	$(call foreach_mod,mod-vet)

install-githook:		## Install githook to verify code before commit
install-githook: .git/hooks/pre-commit

.git/hooks/pre-commit:
	printf "#!/bin/bash\n\nmake vet" > .git/hooks/pre-commit
	chmod 775 .git/hooks/pre-commit


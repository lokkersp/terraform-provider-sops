export CGO_ENABLED = 0
VERSION_TAG = $(shell git describe --tags --match='v*' --always)
RELEASE = $(patsubst v%,%,$(VERSION_TAG))# Remove leading v to comply with Terraform Registry conventions
VERSION_NUMBER = $(shell git describe --tags --match='v*' --always | cut -c 2- )
CROSSBUILD_OS   = linux windows darwin
CROSSBUILD_ARCH = amd64 arm64 # 386
SKIP_OSARCH     = darwin_386 windows_arm64
OSARCH_COMBOS   = $(filter-out $(SKIP_OSARCH),$(foreach os,$(CROSSBUILD_OS),$(addprefix $(os)_,$(CROSSBUILD_ARCH))))
RELEASE_FOLDER  = ~/.terraform.d/plugins/registry.terraform.io/lokkersp/sops/$(VERSION_NUMBER)


default: build

style:
	@echo ">> checking code style"
	! gofmt -d $(shell find . -name '*.go' -print) | grep '^'

vet:
	@echo ">> vetting code"
	go vet ./...

test:
	@echo ">> testing code"
	go test -v ./...

build:
	@echo ">> building binaries"
	go build -o terraform-provider-sops

crossbuild: $(GOPATH)/bin/gox
	@echo ">> cross-building"
	gox -arch="$(CROSSBUILD_ARCH)" -os="$(CROSSBUILD_OS)" -osarch="$(addprefix !,$(subst _,/,$(SKIP_OSARCH)))" \
		-output="binaries/$(VERSION_TAG)/{{.OS}}_{{.Arch}}/terraform-provider-sops_$(VERSION_TAG)"

$(GOPATH)/bin/gox:
	# Need to disable modules for this to not pollute go.mod
	@GO111MODULE=off go get -u github.com/mitchellh/gox

install: crossbuild
	@echo ">> install locally"
	mkdir -p $(RELEASE_FOLDER)
	set -e; for OSARCH in $(OSARCH_COMBOS); do \
		cp -Rf binaries/$(VERSION_TAG)/$$OSARCH/ $(RELEASE_FOLDER)/$$OSARCH; \
	done

# ./bin/hub release edit -m "" -a "releases/terraform-provider-sops_v0.6.4_linux_amd64.zip" v0.6.4
# ./bin/hub release edit -m "" -a "releases/terraform-provider-sops_0.6.4_linux_amd64.zip#terraform-provider-sops_0.6.4_linux_amd64.zip" v0.6.4
release: crossbuild bin/hub
	@echo ">> uploading release $(VERSION_TAG)"
	#./bin/hub release create -m $(VERSION_TAG) $(VERSION_TAG)
	mkdir -p releases
	set -e; for OSARCH in $(OSARCH_COMBOS); do \
		zip -j releases/terraform-provider-sops_$(RELEASE)_$$OSARCH.zip binaries/$(VERSION_TAG)/$$OSARCH/terraform-provider-sops_* > /dev/null; \
		./bin/hub release edit -m "$(RELEASE)" -a "releases/terraform-provider-sops_$(RELEASE)_$$OSARCH.zip#terraform-provider-sops_$(RELEASE)_$$OSARCH.zip" $(VERSION_TAG); \
	done
	@echo ">>> generating sha256sums:"
	cd releases; sha256sum *.zip | tee terraform-provider-sops_$(RELEASE)_SHA256SUMS
	cd releases; gpg --detach-sign terraform-provider-sops_$(RELEASE)_SHA256SUMS
	./bin/hub release edit -m "" -a "releases/terraform-provider-sops_$(RELEASE)_SHA256SUMS#terraform-provider-sops_$(RELEASE)_SHA256SUMS" $(VERSION_TAG)
	./bin/hub release edit -m "" -a "releases/terraform-provider-sops_$(RELEASE)_SHA256SUMS.sig#terraform-provider-sops_$(RELEASE)_SHA256SUMS.sig" $(VERSION_TAG)

bin/hub:
	@mkdir -p bin
	curl -sL 'https://github.com/github/hub/releases/download/v2.14.1/hub-linux-amd64-2.14.1.tgz' | \
		tar -xzf - --strip-components 2 -C bin --wildcards '*/bin/hub'

.PHONY: all style vet test build crossbuild release

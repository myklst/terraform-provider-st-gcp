# The name of Terraform custom provider.
CUSTOM_PROVIDER_NAME ?= terraform-provider-st-gcp
# The url of Terraform provider.
CUSTOM_PROVIDER_URL ?= example.local/myklst/st-gcp

.PHONY: install-local-custom-provider
install-local-custom-provider:
	export PROVIDER_LOCAL_PATH='$(CUSTOM_PROVIDER_URL)'
	go install .
	GO_INSTALL_PATH="$$(go env GOPATH)/bin"; \
	HOME_DIR="$$(ls -d ~)"; \
	mkdir -p  $$HOME_DIR/.terraform.d/plugins/$(CUSTOM_PROVIDER_URL)/0.1.0/linux_amd64/; \
	cp $$GO_INSTALL_PATH/$(CUSTOM_PROVIDER_NAME) $$HOME_DIR/.terraform.d/plugins/$(CUSTOM_PROVIDER_URL)/0.1.0/linux_amd64/$(CUSTOM_PROVIDER_NAME)
	unset PROVIDER_LOCAL_PATH

.PHONY: generate-terraform-document
generate-terraform-document:
	go generate ./...

go-fmt:
	goimports -l -w -local "github.com/myklst/terraform-provider-st-gcp/" .

go-lint:
	golangci-lint run

TAG ?= latest

PROJECT_NAME := "ros"
PKG := "github.com/oam-dev/cloud-provider/alibabacloud/$(PROJECT_NAME)"
PKG_LIST := $(shell go list ${PKG}/... | grep -v /vendor/)
GO_FILES := $(shell find . -name '*.go' | grep -v /vendor/ | grep -v _test.go)
CRD_OPTIONS ?= "crd:trivialVersions=true"

.PHONY: build
build:
	docker build -t oamdev/ros:${TAG} .

.PHONY: publish
publish: build
	docker push oamdev/ros:${TAG}

.PHONY: lint
lint: ## Lint Golang files
	@golint -set_exit_status ${PKG_LIST}

.PHONY: vet
vet: ## Run go vet
	@go vet ${PKG_LIST}

.PHONY: test
test: ## Run unittests
	@go test -short ${PKG_LIST}

.PHONY: test-coverage
test-coverage: ## Run tests with coverage
	@go test -short -coverprofile cover.out -covermode=atomic ${PKG_LIST}
	@cat cover.out

.PHONY: build-binary
build-binary: ## Build the binary file
	@go build -i $(PKG)

# Generate manifests e.g. CRD, RBAC etc.
manifests: controller-gen
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./apis/ros.alibabacloud.com/..." output:crd:artifacts:config=charts/ros/crds/

# Generate code
generate: controller-gen
	$(CONTROLLER_GEN) object:headerFile=./hack/boilerplate.go.txt paths=./apis/ros.alibabacloud.com/...
	./hack/update-client-gen.sh

# find or download controller-gen
# download controller-gen if necessary
# make sure use controller-gen v0.2.5
controller-gen:
ifeq (, $(shell which controller-gen))
	go get sigs.k8s.io/controller-tools/cmd/controller-gen@v0.2.5
CONTROLLER_GEN=$(shell go env GOPATH)/bin/controller-gen
else
CONTROLLER_GEN=$(shell which controller-gen)
endif
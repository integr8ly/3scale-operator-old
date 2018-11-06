ORG=integreatly
NAMESPACE=3scale
PROJECT=3scale-operator
SHELL = /bin/bash
TAG = 0.0.3
PKG = github.com/integr8ly/3scale-operator
TEST_DIRS     ?= $(shell sh -c "find $(TOP_SRC_DIRS) -name \\*_test.go -exec dirname {} \\; | sort | uniq")

.PHONY: check-gofmt
check-gofmt:
	diff -u <(echo -n) <(gofmt -d `find . -type f -name '*.go' -not -path "./vendor/*"`)

.PHONY: test-unit
test-unit:
	@echo Running tests:
	go test -v -race -cover ./pkg/...

.PHONY: test
test: check-gofmt test-unit

.PHONY: setup
setup:
	@echo Installing dep
	curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh
	@echo Installing errcheck
	@go get github.com/kisielk/errcheck
	@go get -u github.com/gobuffalo/packr/packr
	@echo setup complete run make build deploy to build and deploy the operator to a local cluster

.PHONY: build-image
build-image: packr compile build packr-clean

.PHONY: build
build:
	operator-sdk build quay.io/${ORG}/${PROJECT}:${TAG}

.phony: push
push:
	docker push quay.io/$(ORG)/$(PROJECT):$(TAG)

.phony: build-and-push
build-and-push: build-image push

.PHONY: run
run:
	operator-sdk up local --namespace=${NAMESPACE} --operator-flags="--resync=10 --log-level=debug"

.PHONY: generate
generate:
	operator-sdk generate k8s

.PHONY: compile
compile:
	go build -o=3scale-operator ./cmd/3scale-operator

.PHONY: packr
packr:
	packr

.PHONY: packr-clean
packr-clean:
	packr clean

.PHONY: check
check: check-gofmt test-unit
	@echo errcheck
	@errcheck -ignoretests $$(go list ./...)
	@echo go vet
	@go vet ./...

.PHONY: install
install: install-crds
	-oc new-project $(NAMESPACE)
	-oc delete limits $(NAMESPACE)-core-resource-limits
	-kubectl create --insecure-skip-tls-verify -f deploy/rbac.yaml -n $(NAMESPACE)

.PHONY: install-crds
install-crds:
	-kubectl create -f deploy/crd.yaml

.PHONY: uninstall
uninstall:
	-kubectl delete role 3scale-operator -n $(NAMESPACE)
	-kubectl delete rolebinding default-account-3scale-operator -n $(NAMESPACE)
	-kubectl delete crd threescales.3scale.net
	-kubectl delete namespace $(NAMESPACE)

.PHONY: create-examples
create-examples:
	-kubectl create -f deploy/cr.yaml -n $(NAMESPACE)

.PHONY: delete-examples
delete-examples:
	-kubectl delete threescales example

.PHONY: deploy
deploy:
	-kubectl create --insecure-skip-tls-verify -f deploy/operator.yaml -n $(NAMESPACE)

ORG=integreatly
NAMESPACE=3scale
PROJECT=3scale-operator
SHELL = /bin/bash
TAG = 0.0.4
PKG = github.com/integr8ly/3scale-operator
TEST_DIRS     ?= $(shell sh -c "find $(TOP_SRC_DIRS) -name \\*_test.go -exec dirname {} \\; | sort | uniq")
COMPILE_TARGET = build/_output/bin/3scale-operator

.PHONY: setup/dep
setup/dep:
	@echo Installing deps
	curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh
	@go get -u github.com/gobuffalo/packr/packr
	@echo setup complete

.PHONY: setup/travis
setup/travis:
	@echo Installing Operator SDK
	@curl -Lo operator-sdk https://github.com/operator-framework/operator-sdk/releases/download/v0.1.1/operator-sdk-v0.1.1-x86_64-linux-gnu && chmod +x operator-sdk && sudo mv operator-sdk /usr/local/bin/

.PHONY: code/run
code/run:
	@operator-sdk up local --namespace=${NAMESPACE} --operator-flags="--resync=10 --log-level=debug"

.PHONY: code/compile
code/compile:
	@packr
	@GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o=$(COMPILE_TARGET) ./cmd/manager
	@packr clean

.PHONY: code/gen
code/gen:
	@operator-sdk generate k8s

.PHONY: code/check
code/check:
	@diff -u <(echo -n) <(gofmt -d `find . -type f -name '*.go' -not -path "./vendor/*"`)

.PHONY: code/fix
code/fix:
	@gofmt -w `find . -type f -name '*.go' -not -path "./vendor/*"`

.PHONY: image/build
image/build:
	@packr
	@operator-sdk build quay.io/${ORG}/${PROJECT}:${TAG}
	@packr clean

.PHONY: image/build/push
image/build/push: image/build
	@docker push quay.io/$(ORG)/$(PROJECT):$(TAG)

.PHONY: test/unit
test/unit:
	@echo Running tests:
	go test -v -race -cover ./pkg/...

.PHONY: test/e2e
test/e2e:
	@echo Running e2e tests:
	-operator-sdk test local ./test/e2e --go-test-flags "-v"

.PHONY: cluster/prepare
cluster/prepare:
	-kubectl create -f deploy/crds/threescale_v1alpha1_threescale_crd.yaml
	-oc new-project $(NAMESPACE)
	-oc delete limits $(NAMESPACE)-core-resource-limits
	-kubectl create --insecure-skip-tls-verify -f deploy/role.yaml -n $(NAMESPACE)
	-kubectl create --insecure-skip-tls-verify -f deploy/role_binding.yaml -n $(NAMESPACE)
	-kubectl create -f deploy/crds/threescale_v1alpha1_threescale_crd.yaml

.PHONY: cluster/deploy
cluster/deploy:
	-kubectl create --insecure-skip-tls-verify -f deploy/operator.yaml -n $(NAMESPACE)

.PHONY: cluster/deploy/remove
cluster/deploy/remove:
	-kubectl delete --insecure-skip-tls-verify -f deploy/operator.yaml -n $(NAMESPACE)

.PHONY: cluster/clean
cluster/clean:
	-kubectl delete role 3scale-operator -n $(NAMESPACE)
	-kubectl delete rolebinding 3scale-operator -n $(NAMESPACE)
	-kubectl delete crd threescales.threescale.net
	-kubectl delete namespace $(NAMESPACE)

.PHONY: cluster/create/examples
cluster/create/examples:
	-kubectl create -f deploy/crds/threescale_v1alpha1_threescale_cr.yaml -n $(NAMESPACE)

.PHONY: cluster/delete/examples
cluster/delete/examples:
	-kubectl delete threescales example-threescale

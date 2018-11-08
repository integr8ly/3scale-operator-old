ARG BASEIMAGE

FROM ${BASEIMAGE}

ADD tmp/_output/bin/3scale-operator-test /usr/local/bin/3scale-operator-test
ARG NAMESPACEDMAN
ADD $NAMESPACEDMAN /namespaced.yaml
ADD tmp/build/go-test.sh /go-test.sh

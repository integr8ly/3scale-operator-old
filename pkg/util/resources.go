package util

import (
	"github.com/operator-framework/operator-sdk/pkg/util/k8sutil"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/gobuffalo/packr"
	"io/ioutil"
	"k8s.io/apimachinery/pkg/util/yaml"
	"strings"
)

func LoadKubernetesResourceFromFile(path string) (runtime.Object, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return LoadKubernetesResourceData(data, path)
}

func LoadKubernetesResourceFromBox(box packr.Box, name string) (runtime.Object, error) {
	strData, err := box.FindString(name)
	if err != nil {
		return nil, err
	}
	data := []byte(strData)

	return LoadKubernetesResourceData(data, name)
}

func LoadKubernetesResourceData(data []byte, filename string) (runtime.Object, error) {
	data, err := jsonIfYaml(data, filename)
	if err != nil {
		return nil, err
	}

	return LoadKubernetesResource(data)
}

func LoadKubernetesResource(jsonData []byte) (runtime.Object, error) {
	u := unstructured.Unstructured{}
	err := u.UnmarshalJSON(jsonData)
	if err != nil {
		return nil, err
	}

	obj, err := k8sutil.RuntimeObjectFromUnstructured(&u)
	return obj, err
}

func jsonIfYaml(source []byte, filename string) ([]byte, error) {
	if strings.HasSuffix(filename, ".yaml") || strings.HasSuffix(filename, ".yml") {
		return yaml.ToJSON(source)
	}
	return source, nil
}

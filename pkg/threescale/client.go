package threescale

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	threescalev1alpha1 "github.com/integr8ly/3scale-operator/pkg/apis/threescale/v1alpha1"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"net/http"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

type Requester interface {
	Do(req *http.Request) (*http.Response, error)
}

// defaultRequester returns a default client for requesting http endpoints
func defaultRequester() Requester {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	c := &http.Client{Transport: transport, Timeout: time.Second * 10}
	return c
}

type Client struct {
	requester Requester
	URL       string
	token     string
}

// T is a generic type for spec resources
type T interface{}

func (c *Client) CreateUser(user *threescalev1alpha1.ThreeScaleUser) error {
	resourcePath := "admin/api/users.json"
	return c.create(user, resourcePath, "users")
}

func (c *Client) GetUser(id int) (*threescalev1alpha1.ThreeScaleApiUser, error) {
	resourcePath := fmt.Sprintf("admin/api/users/%v.json", id)
	result, err := c.get(resourcePath, "users", func(body []byte) (T, error) {
		var user *threescalev1alpha1.ThreeScaleApiUser
		err := json.Unmarshal(body, &user)
		return user, err
	})
	if err != nil {
		return nil, err
	}
	return result.(*threescalev1alpha1.ThreeScaleApiUser), err
}

func (c *Client) UpdateUser(id int, user *threescalev1alpha1.ThreeScaleUser) error {
	resourcePath := fmt.Sprintf("admin/api/users/%v.json", id)
	return c.update(user, resourcePath, "users")
}

func (c *Client) UpdateUserRole(id int, role string) error {
	resourcePath := fmt.Sprintf("admin/api/users/%v/%s.json", id, role)
	return c.update(nil, resourcePath, "users")
}

func (c *Client) ActivateUser(id int) error {
	resourcePath := fmt.Sprintf("admin/api/users/%v/%s.json", id, "activate")
	return c.update(nil, resourcePath, "users")
}

func (c *Client) ListUsers() ([]*threescalev1alpha1.ThreeScaleApiUser, error) {
	resourcePath := "admin/api/users.json"
	result, err := c.list(resourcePath, "users", func(body []byte) (T, error) {
		var users *threescalev1alpha1.ThreeScaleUserList
		err := json.Unmarshal(body, &users)
		return users.Items, err
	})
	if err != nil {
		return nil, err
	}
	return result.([]*threescalev1alpha1.ThreeScaleApiUser), err
}

func (c *Client) CreateAuthProvider(user *threescalev1alpha1.ThreeScaleAuthProvider) error {
	resourcePath := "admin/api/account/authentication_providers.json"
	return c.create(user, resourcePath, "authentication_providers")
}

func (c *Client) ListAuthProviders() ([]*threescalev1alpha1.ThreeScaleApiAuthProvider, error) {
	resourcePath := "admin/api/account/authentication_providers.json"
	result, err := c.list(resourcePath, "authentication_providers", func(body []byte) (T, error) {
		var provider *threescalev1alpha1.ThreeScaleAuthProviderList
		err := json.Unmarshal(body, &provider)
		return provider.Items, err
	})
	if err != nil {
		return nil, err
	}
	return result.([]*threescalev1alpha1.ThreeScaleApiAuthProvider), err
}

// Generic create function for creating new ThreeScale resources
func (c *Client) create(obj T, resourcePath, resourceName string) error {
	reqParams := fmt.Sprintf("access_token=%s", c.token)

	jsonValue, err := json.Marshal(obj)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(
		"POST",
		fmt.Sprintf("%s/%s?%s", c.URL, resourcePath, reqParams),
		bytes.NewBuffer(jsonValue),
	)
	if err != nil {
		return errors.Wrapf(err, "error creating POST %s request", resourceName)
	}

	req.Header.Set("Content-Type", "application/json")
	res, err := c.requester.Do(req)

	if err != nil {
		return errors.Wrapf(err, "error performing POST %s request", resourceName)
	}
	defer res.Body.Close()

	logrus.Debugf("response status: %v, %v", res.StatusCode, res.Status)
	if res.StatusCode != 201 {
		return fmt.Errorf("failed to create %s: (%d) %s", resourceName, res.StatusCode, res.Status)
	}

	logrus.Debugf("response:", res)
	return nil
}

// Generic get function for returning a ThreeScale resource
func (c *Client) get(resourcePath, resourceName string, unMarshalFunc func(body []byte) (T, error)) (T, error) {
	reqParams := fmt.Sprintf("access_token=%s", c.token)
	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf("%s/%s?%s", c.URL, resourcePath, reqParams),
		nil,
	)
	if err != nil {
		return nil, errors.Wrapf(err, "error creating GET %s request", resourceName)
	}

	res, err := c.requester.Do(req)
	if err != nil {
		return nil, errors.Wrapf(err, "error performing GET %s request", resourceName)
	}

	logrus.Debugf("response status: %v, %v", res.StatusCode, res.Status)
	if res.StatusCode != 200 {
		return nil, fmt.Errorf("failed to GET %s: (%d) %s", resourceName, res.StatusCode, res.Status)
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, errors.Wrapf(err, "error reading %s GET response", resourceName)
	}

	logrus.Debugf("%s GET: %+v\n", resourceName, string(body))

	obj, err := unMarshalFunc(body)
	if err != nil {
		return nil, err
	}
	logrus.Debugf("%s GET= %#v", resourceName, obj)

	return obj, nil
}

// Generic put function for updating ThreeScale resources
func (c *Client) update(obj T, resourcePath, resourceName string) error {
	reqParams := fmt.Sprintf("access_token=%s", c.token)
	jsonValue, err := json.Marshal(obj)
	if err != nil {
		return nil
	}

	req, err := http.NewRequest(
		"PUT",
		fmt.Sprintf("%s/%s?%s", c.URL, resourcePath, reqParams),
		bytes.NewBuffer(jsonValue),
	)
	if err != nil {
		return errors.Wrapf(err, "error creating UPDATE %s request", resourceName)
	}

	req.Header.Set("Content-Type", "application/json")
	res, err := c.requester.Do(req)
	if err != nil {
		return errors.Wrapf(err, "error performing UPDATE %s request", resourceName)
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode > 299 {
		return fmt.Errorf("failed to UPDATE %s: (%d) %s", resourceName, res.StatusCode, res.Status)
	}

	logrus.Debugf("response:", res)
	return nil
}

// Generic list function for listing resources
func (c *Client) list(resourcePath, resourceName string, unMarshalListFunc func(body []byte) (T, error)) (T, error) {
	reqParams := fmt.Sprintf("access_token=%s", c.token)
	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf("%s/%s?%s", c.URL, resourcePath, reqParams),
		nil,
	)
	if err != nil {
		return nil, errors.Wrapf(err, "error creating LIST %s request", resourceName)
	}

	res, err := c.requester.Do(req)
	if err != nil {
		return nil, errors.Wrapf(err, "error performing LIST %s request", resourceName)
	}
	defer res.Body.Close()

	logrus.Debugf("response status: %v, %v", res.StatusCode, res.Status)
	if res.StatusCode < 200 || res.StatusCode > 299 {
		return nil, fmt.Errorf("failed to LIST %s: (%d) %s", resourceName, res.StatusCode, res.Status)
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, errors.Wrapf(err, "error reading %s LIST response", resourceName)
	}

	logrus.Debugf("%s LIST: %+v\n", resourceName, string(body))

	objs, err := unMarshalListFunc(body)
	if err != nil {
		return nil, err
	}
	logrus.Debugf("%s LIST= %#v", resourceName, objs)

	return objs, nil
}

type ThreeScaleInterface interface {
	CreateUser(user *threescalev1alpha1.ThreeScaleUser) error
	GetUser(id int) (*threescalev1alpha1.ThreeScaleApiUser, error)
	UpdateUser(id int, user *threescalev1alpha1.ThreeScaleUser) error
	UpdateUserRole(id int, role string) error
	ActivateUser(id int) error
	ListUsers() ([]*threescalev1alpha1.ThreeScaleApiUser, error)

	CreateAuthProvider(authProvider *threescalev1alpha1.ThreeScaleAuthProvider) error
	//GetAuthProvider(id int) (*threescalev1alpha1.ThreeScaleAuthProvider, error)
	//UpdateAuthProvider(id int, authProvider *threescalev1alpha1.ThreeScaleAuthProvider) error
	ListAuthProviders() ([]*threescalev1alpha1.ThreeScaleApiAuthProvider, error)
}

type ThreeScaleClientFactory interface {
	AuthenticatedClient(kc threescalev1alpha1.ThreeScale) (ThreeScaleInterface, error)
}

type ThreeScaleFactory struct {
	Client client.Client
}

// AuthenticatedClient returns an authenticated client for requesting endpoints from the ThreeScale api
func (tsf *ThreeScaleFactory) AuthenticatedClient(ts threescalev1alpha1.ThreeScaleTenant) (ThreeScaleInterface, error) {
	adminCreds := &v1.Secret{}
	err := tsf.Client.Get(context.TODO(), types.NamespacedName{Name: ts.Spec.AdminCredentials, Namespace: ts.Namespace}, adminCreds)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get the admin credentials")
	}
	token := string(adminCreds.Data["ADMIN_ACCESS_TOKEN"])
	url := string(adminCreds.Data["ADMIN_URL"])
	client := &Client{
		URL:       url,
		token:     token,
		requester: defaultRequester(),
	}
	return client, nil
}

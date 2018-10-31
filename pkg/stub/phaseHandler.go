package stub

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/integr8ly/3scale-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/3scale-operator/pkg/clients/openshift"
	"github.com/integr8ly/3scale-operator/pkg/threescale"
	"github.com/operator-framework/operator-sdk/pkg/util/k8sutil"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	errors2 "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"reflect"
	"strings"
)

type phaseHandler struct {
	k8sClient                    kubernetes.Interface
	osClient                     *openshift.ClientFactory
	tsFactory                    *threescale.ThreeScaleFactory
	dynamicResourceClientFactory func(apiVersion, kind, namespace string) (dynamic.ResourceInterface, string, error)
}

func NewPhaseHandler(k8sClient kubernetes.Interface, osClient *openshift.ClientFactory, tsFactory *threescale.ThreeScaleFactory, dynamicResourceClientFactory func(apiVersion, kind, namespace string) (dynamic.ResourceInterface, string, error)) *phaseHandler {
	return &phaseHandler{
		k8sClient:                    k8sClient,
		osClient:                     osClient,
		tsFactory:                    tsFactory,
		dynamicResourceClientFactory: dynamicResourceClientFactory,
	}
}

func (ph *phaseHandler) Initialise(obj *v1alpha1.ThreeScale) (*v1alpha1.ThreeScale, error) {
	logrus.Info("Initialise")
	// copy state and modify return state
	tsState := obj.DeepCopy()
	// fill in any defaults that are not set
	tsState.Defaults()
	// validate
	if err := tsState.Validate(); err != nil {
		return nil, errors.Wrap(err, "validation failed")
	}

	tsState.Status.Phase = v1alpha1.PhaseProvisionCredentials
	return tsState, nil
}

func (ph *phaseHandler) Credentials(obj *v1alpha1.ThreeScale) (*v1alpha1.ThreeScale, error) {
	logrus.Info("Credentials")
	// copy state and modify return state
	ts := obj.DeepCopy()
	// fill in any defaults that are not set
	ts.Defaults()

	//Create admin access token
	adminSecret, err := ph.createOpaqueSecret("admin-credentials", "ADMIN_ACCESS_TOKEN", ts)
	if err != nil {
		return ts, errors.Wrap(err, "failed to create admin token")
	}
	ts.Spec.AdminCredentials = adminSecret.GetName()
	//Create master access token
	masterSecret, err := ph.createOpaqueSecret("3scale-master-access-token", "MASTER_ACCESS_TOKEN", ts)
	if err != nil {
		return ts, errors.Wrap(err, "failed to create master access token")
	}
	ts.Spec.MasterCredentials = masterSecret.GetName()

	ts.Status.Phase = v1alpha1.PhaseReconcileThreescale
	return ts, nil
}

func (ph *phaseHandler) ReconcileThreeScale(obj *v1alpha1.ThreeScale) (*v1alpha1.ThreeScale, error) {
	logrus.Info("ReconcileThreeScale")
	// copy state and modify return state
	ts := obj.DeepCopy()
	// fill in any defaults that are not set
	ts.Defaults()

	ts, err := ph.InstallThreeScale(ts)
	if err != nil {
		return nil, errors.Wrap(err, "error provisioning threescale")
	}
	ts, err = ph.CheckInstallResourcesReady(ts)
	if err != nil {
		return nil, errors.Wrap(err, "error checking pods ready")
	}

	if ts.Status.Ready {
		tsClient, err := ph.tsFactory.AuthenticatedClient(*ts)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get authenticated client for threescale")
		}

		ts, err = ph.ReconcileAuthProviders(ts, tsClient)
		if err != nil {
			return nil, errors.Wrap(err, "error reconciling auth providers")
		}
		ts, err = ph.ReconcileUsers(ts, tsClient)
		if err != nil {
			return nil, errors.Wrap(err, "error reconciling users")
		}
	}

	return ts, nil
}

func (ph *phaseHandler) InstallThreeScale(ts *v1alpha1.ThreeScale) (*v1alpha1.ThreeScale, error) {
	logrus.Info("InstallThreeScale")
	//Set params
	decodedParams := map[string]string{}
	adminCreds, err := ph.k8sClient.CoreV1().Secrets(ts.Namespace).Get(ts.Spec.AdminCredentials, metav1.GetOptions{})
	masterCreds, err := ph.k8sClient.CoreV1().Secrets(ts.Namespace).Get(ts.Spec.MasterCredentials, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get the secret for the admin credentials")
	}
	for k, v := range adminCreds.Data {
		decodedParams[k] = string(v)
	}
	for k, v := range masterCreds.Data {
		decodedParams[k] = string(v)
	}
	if ts.Spec.RWXStorageClass != "" {
		decodedParams["RWX_STORAGE_CLASS"] = string(ts.Spec.RWXStorageClass)
	}
	if ts.Spec.WildcardPolicy != "" {
		decodedParams["WILDCARD_POLICY"] = string(ts.Spec.WildcardPolicy)
	}
	if ts.Spec.MysqlPvcSize != "" {
		decodedParams["MYSQL_PVC_SIZE"] = string(ts.Spec.MysqlPvcSize)
	}
	if ts.Spec.AdminUsername != "" {
		decodedParams["ADMIN_USERNAME"] = string(ts.Spec.AdminUsername)
	}
	if ts.Spec.AdminEmail != "" {
		decodedParams["ADMIN_EMAIL"] = string(ts.Spec.AdminEmail)
	}
	decodedParams["WILDCARD_DOMAIN"] = string(ts.Spec.RouteSuffix)
	//Set params

	objects, err := GetInstallResourcesAsRuntimeObjects(ts, decodedParams)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get runtime objects during provision")
	}
	for _, o := range objects {
		gvk := o.GetObjectKind().GroupVersionKind()
		apiVersion, kind := gvk.ToAPIVersionAndKind()
		resourceClient, _, err := ph.dynamicResourceClientFactory(apiVersion, kind, ts.Namespace)
		if err != nil {
			return nil, errors.New(fmt.Sprintf("failed to get resource client: %v", err))
		}

		unstructObj, err := k8sutil.UnstructuredFromRuntimeObject(o)
		ownerRefs := unstructObj.GetOwnerReferences()
		ownerRefs = append(ownerRefs, *NewOwnerRef(ts))
		unstructObj.SetOwnerReferences(ownerRefs)

		if err != nil {
			return nil, errors.Wrap(err, "failed to turn runtime object "+o.GetObjectKind().GroupVersionKind().String()+" into unstructured object during provision")
		}
		unstructObj, err = resourceClient.Create(unstructObj)
		if err != nil && !errors2.IsAlreadyExists(err) {
			return nil, errors.Wrap(err, "failed to create object during provision with kind "+o.GetObjectKind().GroupVersionKind().String())
		}
	}

	return ts, nil
}

func (ph *phaseHandler) CheckInstallResourcesReady(ts *v1alpha1.ThreeScale) (*v1alpha1.ThreeScale, error) {
	logrus.Info("CheckInstallResourcesReady")
	podList, err := ph.k8sClient.CoreV1().Pods(ts.Namespace).List(metav1.ListOptions{
		IncludeUninitialized: false,
	})

	ts.Status.Ready = false

	if err != nil || len(podList.Items) == 0 {
		return ts, nil
	}
	for _, pod := range podList.Items {
		for _, condition := range pod.Status.Conditions {
			if condition.Type == "Ready" && condition.Status != "True" {
				return ts, nil
			}
		}
	}

	osClient, err := ph.osClient.RouteClient()

	adminRoute, err := osClient.Routes(ts.Namespace).Get("system-provider-admin-route", metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get admin route")
	}
	protocol := "https"
	if adminRoute.Spec.TLS == nil {
		protocol = "http"
	}
	adminUrl := fmt.Sprintf("%v://%v", protocol, adminRoute.Spec.Host)

	adminCreds, err := ph.k8sClient.CoreV1().Secrets(ts.Namespace).Get(ts.Spec.AdminCredentials, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get the secret for the admin credentials")
	}
	adminCreds.Data["ADMIN_URL"] = []byte(adminUrl)
	if _, err = ph.k8sClient.CoreV1().Secrets(ts.Namespace).Update(adminCreds); err != nil {
		return nil, errors.Wrap(err, "could not update admin credentials")
	}

	ts.Status.Ready = true
	return ts, nil
}

func (ph *phaseHandler) ReconcileAuthProviders(ts *v1alpha1.ThreeScale, tsClient threescale.ThreeScaleInterface) (*v1alpha1.ThreeScale, error) {
	logrus.Info("ReconcileAuthProviders")
	apiAuthProviders, err := tsClient.ListAuthProviders()
	if err != nil {
		return nil, errors.Wrap(err, "error retrieving auth providers from threescale")
	}
	tsAuthProviders := map[string]*v1alpha1.ThreeScaleAuthProvider{}
	for i := range apiAuthProviders {
		tsAuthProviders[apiAuthProviders[i].AuthProvider.Name] = apiAuthProviders[i].AuthProvider
	}

	specAuthProviders := map[string]*v1alpha1.ThreeScaleAuthProvider{}
	for i := range ts.Spec.AuthProviders {
		specAuthProviders[ts.Spec.AuthProviders[i].Name] = &ts.Spec.AuthProviders[i]
	}

	authproviderPairsList := map[string]*v1alpha1.ThreeScaleAuthProviderPair{}
	for k, _ := range specAuthProviders {
		provider := specAuthProviders[k]
		authproviderPairsList[provider.Name] = &v1alpha1.ThreeScaleAuthProviderPair{
			SpecAuthProvider: provider,
			TsAuthProvider:   tsAuthProviders[provider.Name],
		}
	}

	for i := range authproviderPairsList {
		err := reconcileAuthProvider(authproviderPairsList[i].TsAuthProvider, authproviderPairsList[i].SpecAuthProvider, tsClient)
		if err != nil {
			return nil, err
		}
	}

	return ts, nil
}

func reconcileAuthProvider(tsAuthProvider, specAuthProvider *v1alpha1.ThreeScaleAuthProvider, tsClient threescale.ThreeScaleInterface) error {
	if tsAuthProvider == nil {
		logrus.Infof("create auth provider: %s", specAuthProvider.Name)
		err := tsClient.CreateAuthProvider(specAuthProvider)
		if err != nil {
			return err
		}
	} else {
		//ToDo implement update
	}
	return nil
}

func (ph *phaseHandler) ReconcileUsers(ts *v1alpha1.ThreeScale, tsClient threescale.ThreeScaleInterface) (*v1alpha1.ThreeScale, error) {
	logrus.Info("ReconcileUsers")

	apiUsers, err := tsClient.ListUsers()
	if err != nil {
		return nil, errors.Wrap(err, "error retrieving apiUsers from threescale")
	}
	tsUsers := map[string]*v1alpha1.ThreeScaleUser{}
	for i := range apiUsers {
		tsUsers[apiUsers[i].User.UserName] = apiUsers[i].User
	}

	specUsers := map[string]*v1alpha1.ThreeScaleUser{}
	for i := range ts.Spec.Users {
		specUsers[ts.Spec.Users[i].UserName] = &ts.Spec.Users[i]
	}

	if ts.Spec.SeedUsers.Count != 0 {
		for i := 1; i <= ts.Spec.SeedUsers.Count; i++ {
			username := fmt.Sprintf(ts.Spec.SeedUsers.NameFormat, i)
			if specUsers[username] == nil {
				evalUser := &v1alpha1.ThreeScaleUser{
					Email:    fmt.Sprintf(ts.Spec.SeedUsers.EmailFormat, i),
					UserName: username,
					Password: ts.Spec.SeedUsers.Password,
					Role:     ts.Spec.SeedUsers.Role,
				}
				ts.Spec.Users = append(ts.Spec.Users, *evalUser)
				specUsers[username] = evalUser
			}
		}
	}

	userPairsList := map[string]*v1alpha1.ThreeScaleUserPair{}
	for k, _ := range specUsers {
		user := specUsers[k]
		userPairsList[user.UserName] = &v1alpha1.ThreeScaleUserPair{
			SpecUser: user,
			TsUser:   tsUsers[user.UserName],
		}
	}

	for i := range userPairsList {
		err := reconcileUser(userPairsList[i].TsUser, userPairsList[i].SpecUser, tsClient)
		if err != nil {
			return nil, err
		}
	}

	return ts, nil
}

func reconcileUser(tsUser, specUser *v1alpha1.ThreeScaleUser, tsClient threescale.ThreeScaleInterface) error {
	if tsUser == nil {
		logrus.Infof("create user: %s", specUser.UserName)
		err := tsClient.CreateUser(specUser)
		if err != nil {
			return err
		}
	} else {
		tsUser.Password = specUser.Password
		if !reflect.DeepEqual(tsUser, specUser) {
			logrus.Infof("update user: %s", specUser.UserName)
			specUser.ID = tsUser.ID
			err := tsClient.UpdateUser(specUser.ID, specUser)
			if err != nil {
				return err
			}
			if specUser.Role != "" && specUser.Role != tsUser.Role {
				logrus.Infof("update user role: %s, %s", specUser.UserName, specUser.Role)
				err = tsClient.UpdateUserRole(specUser.ID, specUser.Role)
				if err != nil {
					return err
				}
			} else {
				specUser.Role = tsUser.Role
			}
			specUser.State = tsUser.State
		}

		if specUser.State == "pending" {
			logrus.Infof("activate user: %s", specUser.UserName)
			err := tsClient.ActivateUser(specUser.ID)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (ph *phaseHandler) createOpaqueSecret(name, key string, owner *v1alpha1.ThreeScale) (*v1.Secret, error) {
	token, err := GeneratePassword()
	if err != nil {
		return nil, err
	}
	data := map[string][]byte{key: []byte(token)}
	credentialsSecret := &v1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: owner.Namespace,
			Name:      name,
			OwnerReferences: []metav1.OwnerReference{
				*NewOwnerRef(owner),
			},
		},
		Data: data,
		Type: "Opaque",
	}
	secret, err := ph.k8sClient.CoreV1().Secrets(owner.Namespace).Create(credentialsSecret)
	if err != nil && !errors2.IsAlreadyExists(err) {
		return nil, err
	}

	return secret, nil
}

func NewOwnerRef(owner *v1alpha1.ThreeScale) *metav1.OwnerReference {
	return metav1.NewControllerRef(owner, schema.GroupVersionKind{
		Group:   v1alpha1.SchemeGroupVersion.Group,
		Version: v1alpha1.SchemeGroupVersion.Version,
		Kind:    owner.Kind,
	})
}

func GeneratePassword() (string, error) {
	generatedPassword, err := uuid.NewRandom()
	if err != nil {
		return "", errors.Wrap(err, "error generating password")
	}
	return strings.Replace(generatedPassword.String(), "-", "", 10), nil
}

package threescale

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/integr8ly/3scale-operator/pkg/threescale"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/google/uuid"
	openshift "github.com/integr8ly/3scale-operator/pkg/apis/openshift/client"
	threescalev1alpha1 "github.com/integr8ly/3scale-operator/pkg/apis/threescale/v1alpha1"
	routev1 "github.com/openshift/api/route/v1"
	k8errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// Add creates a new ThreeScale Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileThreeScale{
		client:    mgr.GetClient(),
		scheme:    mgr.GetScheme(),
		config:    mgr.GetConfig(),
		tsFactory: &threescale.ThreeScaleFactory{Client: mgr.GetClient()},
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("threescale-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource ThreeScale
	err = c.Watch(&source.Kind{Type: &threescalev1alpha1.ThreeScale{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileThreeScale{}

// ReconcileThreeScale reconciles a ThreeScale object
type ReconcileThreeScale struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client    client.Client
	scheme    *runtime.Scheme
	config    *rest.Config
	tsFactory *threescale.ThreeScaleFactory
}

// Reconcile reads that state of the cluster for a ThreeScale object and makes changes based on the state read
// and what is in the ThreeScale.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileThreeScale) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	log.Printf("Reconciling ThreeScale %s/%s\n", request.Namespace, request.Name)

	// Fetch the ThreeScale instance
	instance := &threescalev1alpha1.ThreeScale{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if k8errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	switch instance.Status.Phase {
	case threescalev1alpha1.NoPhase:
		tsState, err := r.Initialise(instance)
		if err != nil {
			return reconcile.Result{}, errors.Wrap(err, "failed to init resource")
		}
		return reconcile.Result{Requeue: true}, r.client.Update(context.TODO(), tsState)
	case threescalev1alpha1.PhaseProvisionCredentials:
		tsState, err := r.Credentials(instance)
		if err != nil {
			return reconcile.Result{}, errors.Wrap(err, "phase provision credentials failed")
		}
		return reconcile.Result{Requeue: true}, r.client.Update(context.TODO(), tsState)
	case threescalev1alpha1.PhaseReconcileThreescale:
		tsState, err := r.ReconcileThreeScale(instance)
		if err != nil {
			log.Errorf("phase reconcile threescale failed: %+v", err)
			return reconcile.Result{}, errors.Wrap(err, "phase reconcile threescale failed")
		}
		resyncDuration := time.Second * time.Duration(60)
		return reconcile.Result{Requeue: true, RequeueAfter: resyncDuration}, r.client.Update(context.TODO(), tsState)
	}

	return reconcile.Result{Requeue: true}, nil
}

func (r *ReconcileThreeScale) Initialise(obj *threescalev1alpha1.ThreeScale) (*threescalev1alpha1.ThreeScale, error) {
	log.Info("Initialise")
	// copy state and modify return state
	tsState := obj.DeepCopy()
	// fill in any defaults that are not set
	tsState.Defaults()
	// validate
	if err := tsState.Validate(); err != nil {
		return nil, errors.Wrap(err, "validation failed")
	}

	tsState.Status.Version = threescalev1alpha1.ThreescaleVersion
	tsState.Status.Phase = threescalev1alpha1.PhaseProvisionCredentials
	return tsState, nil
}

func (r *ReconcileThreeScale) Credentials(obj *threescalev1alpha1.ThreeScale) (*threescalev1alpha1.ThreeScale, error) {
	log.Info("Credentials")
	// copy state and modify return state
	ts := obj.DeepCopy()
	// fill in any defaults that are not set
	ts.Defaults()

	//Create admin access token
	adminSecret, err := r.createOpaqueSecret("admin-credentials", "ADMIN_ACCESS_TOKEN", ts)
	if err != nil {
		return ts, errors.Wrap(err, "failed to create admin token")
	}
	ts.Spec.AdminCredentials = adminSecret.GetName()
	//Create master access token
	masterSecret, err := r.createOpaqueSecret("3scale-master-access-token", "MASTER_ACCESS_TOKEN", ts)
	if err != nil {
		return ts, errors.Wrap(err, "failed to create master access token")
	}
	ts.Spec.MasterCredentials = masterSecret.GetName()

	ts.Status.Phase = threescalev1alpha1.PhaseReconcileThreescale
	return ts, nil
}

func (r *ReconcileThreeScale) createOpaqueSecret(name, key string, owner *threescalev1alpha1.ThreeScale) (*v1.Secret, error) {
	token, err := GeneratePassword()
	if err != nil {
		return nil, err
	}
	data := map[string][]byte{key: []byte(token)}
	secret := &v1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: owner.Namespace,
			Name:      name,
		},
		Data: data,
		Type: "Opaque",
	}

	if err := controllerutil.SetControllerReference(owner, secret, r.scheme); err != nil {
		return nil, err
	}

	found := &v1.Secret{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: secret.Name, Namespace: secret.Namespace}, found)
	if err != nil && k8errors.IsNotFound(err) {
		log.Infof("Creating new secret %s/%s\n", secret.Namespace, secret.Name)
		err = r.client.Create(context.TODO(), secret)
		if err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	} else {
		secret = found
	}
	return secret, nil
}

func (r *ReconcileThreeScale) ReconcileThreeScale(obj *threescalev1alpha1.ThreeScale) (*threescalev1alpha1.ThreeScale, error) {
	log.Info("ReconcileThreeScale")
	// copy state and modify return state
	ts := obj.DeepCopy()
	// fill in any defaults that are not set
	ts.Defaults()

	ts, err := r.InstallThreeScale(ts)
	if err != nil {
		return nil, errors.Wrap(err, "error provisioning threescale")
	}
	log.Info("Create Resources: done")
	ts, err = r.CheckInstallResourcesReady(ts)
	if err != nil {
		return nil, errors.Wrap(err, "error checking resources ready")
	}

	if !ts.Status.Ready {
		log.Info("Resources Ready: no")
		return ts, nil
	}
	log.Info("Resources Ready: yes")

	tsClient, err := r.tsFactory.AuthenticatedClient(*ts)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get authenticated client for threescale")
	}

	ts, err = r.ReconcileAuthProviders(ts, tsClient)
	if err != nil {
		return nil, errors.Wrap(err, "error reconciling auth providers")
	}
	log.Info("Reconcile Authentication Providers: done")
	ts, err = r.ReconcileUsers(ts, tsClient)
	if err != nil {
		return nil, errors.Wrap(err, "error reconciling users")
	}
	log.Info("Reconcile Users: done")

	return ts, nil
}

func (r *ReconcileThreeScale) InstallThreeScale(ts *threescalev1alpha1.ThreeScale) (*threescalev1alpha1.ThreeScale, error) {
	// Set params
	decodedParams := map[string]string{}
	adminCreds := &v1.Secret{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: ts.Spec.AdminCredentials, Namespace: ts.Namespace}, adminCreds)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get the secret for the admin credentials")
	}
	masterCreds := &v1.Secret{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: ts.Spec.MasterCredentials, Namespace: ts.Namespace}, masterCreds)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get the secret for the master credentials")
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

	objects, err := r.GetInstallResourcesAsRuntimeObjects(ts, decodedParams)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get runtime objects during provision")
	}

	for _, o := range objects {
		uo, err := openshift.UnstructuredFromRuntimeObject(o)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get runtime object")
		}

		uo.SetNamespace(ts.Namespace)
		if err := controllerutil.SetControllerReference(ts, uo, r.scheme); err != nil {
			return nil, err
		}

		err = r.client.Create(context.TODO(), uo)
		if err != nil && !k8errors.IsAlreadyExists(err) {
			return nil, errors.Wrap(err, "failed to create object during provision with kind "+o.GetObjectKind().GroupVersionKind().String())
		}
	}

	return ts, nil
}

func (r *ReconcileThreeScale) CheckInstallResourcesReady(ts *threescalev1alpha1.ThreeScale) (*threescalev1alpha1.ThreeScale, error) {

	opts := &client.ListOptions{}
	opts.InNamespace(ts.Namespace)

	podList := &v1.PodList{}
	err := r.client.List(context.TODO(), opts, podList)

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

	adminRoute := &routev1.Route{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: "system-provider-admin", Namespace: ts.Namespace}, adminRoute)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get admin route")
	}

	protocol := "https"
	if adminRoute.Spec.TLS == nil {
		protocol = "http"
	}
	adminUrl := fmt.Sprintf("%v://%v", protocol, adminRoute.Spec.Host)

	adminCreds := &v1.Secret{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: ts.Spec.AdminCredentials, Namespace: ts.Namespace}, adminCreds)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get the secret for the admin credentials")
	}
	adminCreds.Data["ADMIN_URL"] = []byte(adminUrl)

	if err = r.client.Update(context.TODO(), adminCreds); err != nil {
		return nil, errors.Wrap(err, "could not update admin credentials")
	}

	ts.Status.Ready = true
	return ts, nil
}

func (r *ReconcileThreeScale) ReconcileAuthProviders(ts *threescalev1alpha1.ThreeScale, tsClient threescale.ThreeScaleInterface) (*threescalev1alpha1.ThreeScale, error) {
	apiAuthProviders, err := tsClient.ListAuthProviders()
	if err != nil {
		return nil, errors.Wrap(err, "error retrieving auth providers from threescale")
	}
	tsAuthProviders := map[string]*threescalev1alpha1.ThreeScaleAuthProvider{}
	for i := range apiAuthProviders {
		tsAuthProviders[apiAuthProviders[i].AuthProvider.Name] = apiAuthProviders[i].AuthProvider
	}

	specAuthProviders := map[string]*threescalev1alpha1.ThreeScaleAuthProvider{}
	for i := range ts.Spec.AuthProviders {
		specAuthProviders[ts.Spec.AuthProviders[i].Name] = &ts.Spec.AuthProviders[i]
	}

	authproviderPairsList := map[string]*threescalev1alpha1.ThreeScaleAuthProviderPair{}
	for k, _ := range specAuthProviders {
		provider := specAuthProviders[k]
		authproviderPairsList[provider.Name] = &threescalev1alpha1.ThreeScaleAuthProviderPair{
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

func reconcileAuthProvider(tsAuthProvider, specAuthProvider *threescalev1alpha1.ThreeScaleAuthProvider, tsClient threescale.ThreeScaleInterface) error {
	if tsAuthProvider == nil {
		log.Infof("create auth provider: %s", specAuthProvider.Name)
		err := tsClient.CreateAuthProvider(specAuthProvider)
		if err != nil {
			return err
		}
	} else {
		//ToDo implement update
	}
	return nil
}

func (r *ReconcileThreeScale) ReconcileUsers(ts *threescalev1alpha1.ThreeScale, tsClient threescale.ThreeScaleInterface) (*threescalev1alpha1.ThreeScale, error) {
	apiUsers, err := tsClient.ListUsers()
	if err != nil {
		return nil, errors.Wrap(err, "error retrieving apiUsers from threescale")
	}
	tsUsers := map[string]*threescalev1alpha1.ThreeScaleUser{}
	for i := range apiUsers {
		tsUsers[apiUsers[i].User.UserName] = apiUsers[i].User
	}

	specUsers := map[string]*threescalev1alpha1.ThreeScaleUser{}
	for i := range ts.Spec.Users {
		specUsers[ts.Spec.Users[i].UserName] = &ts.Spec.Users[i]
	}

	if ts.Spec.SeedUsers.Count != 0 {
		for i := 1; i <= ts.Spec.SeedUsers.Count; i++ {
			username := fmt.Sprintf(ts.Spec.SeedUsers.NameFormat, i)
			if specUsers[username] == nil {
				evalUser := &threescalev1alpha1.ThreeScaleUser{
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

	userPairsList := map[string]*threescalev1alpha1.ThreeScaleUserPair{}
	for k, _ := range specUsers {
		user := specUsers[k]
		userPairsList[user.UserName] = &threescalev1alpha1.ThreeScaleUserPair{
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

func reconcileUser(tsUser, specUser *threescalev1alpha1.ThreeScaleUser, tsClient threescale.ThreeScaleInterface) error {
	if tsUser == nil {
		log.Infof("create user: %s", specUser.UserName)
		return tsClient.CreateUser(specUser)
	}
	tsUser.Password = specUser.Password
	if !reflect.DeepEqual(tsUser, specUser) {
		log.Infof("update user: %s", specUser.UserName)
		specUser.ID = tsUser.ID
		err := tsClient.UpdateUser(specUser.ID, specUser)
		if err != nil {
			return err
		}
		if specUser.Role != "" && specUser.Role != tsUser.Role {
			log.Infof("update user role: %s, %s", specUser.UserName, specUser.Role)
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
		log.Infof("activate user: %s", specUser.UserName)
		return tsClient.ActivateUser(specUser.ID)
	}

	return nil
}

func GeneratePassword() (string, error) {
	generatedPassword, err := uuid.NewRandom()
	if err != nil {
		return "", errors.Wrap(err, "error generating password")
	}
	return strings.Replace(generatedPassword.String(), "-", "", 10), nil
}

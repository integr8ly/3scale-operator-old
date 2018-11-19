package threescaletenant

import (
	"context"
	"fmt"
	"github.com/integr8ly/3scale-operator/pkg/threescale"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"time"

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

// Add creates a new ThreeScaleTenant Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileThreeScaleTenant{
		client:    mgr.GetClient(),
		scheme:    mgr.GetScheme(),
		tsFactory: &threescale.ThreeScaleFactory{Client: mgr.GetClient()},
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("threescaletenant-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource ThreeScaleTenant
	err = c.Watch(&source.Kind{Type: &threescalev1alpha1.ThreeScaleTenant{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileThreeScaleTenant{}

// ReconcileThreeScaleTenant reconciles a ThreeScaleTenant object
type ReconcileThreeScaleTenant struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client    client.Client
	scheme    *runtime.Scheme
	tsFactory *threescale.ThreeScaleFactory
}

// Reconcile reads that state of the cluster for a ThreeScaleTenant object and makes changes based on the state read
// and what is in the ThreeScaleTenant.Spec
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileThreeScaleTenant) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	log.Printf("Reconciling ThreeScaleTenant %s/%s\n", request.Namespace, request.Name)

	// Fetch the ThreeScaleTenant instance
	instance := &threescalev1alpha1.ThreeScaleTenant{}
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

	tst := instance.DeepCopy()
	// fill in any defaults that are not set
	tst.Defaults()

	resyncDuration := time.Second * time.Duration(10)

	ts := &threescalev1alpha1.ThreeScale{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: tst.Spec.ThreeScaleName, Namespace: tst.Namespace}, ts)
	if err != nil {
		log.Debugf("failed to get threescale resource (%s)", tst.Spec.ThreeScaleName)
		return reconcile.Result{Requeue: true, RequeueAfter: resyncDuration}, errors.Wrap(err, "failed to get threescale resource")
	}

	if err := controllerutil.SetControllerReference(ts, tst, r.scheme); err != nil {
		log.Debugf("failed to set owner (%s)", tst.Spec.ThreeScaleName)
		return reconcile.Result{Requeue: true, RequeueAfter: resyncDuration}, errors.Wrap(err, "failed to set owner")
	}

	if !ts.Status.Ready {
		log.Debugf("threescale resource (%s) not ready yet", tst.Spec.ThreeScaleName)
		return reconcile.Result{Requeue: true, RequeueAfter: resyncDuration}, nil
	}

	tst, err = r.CreateTenant(ts, tst)
	if err != nil {
		return reconcile.Result{}, errors.Wrap(err, "error creating tenant")
	}

	tst, err = r.ReconcileRoutes(tst)
	if err != nil {
		return reconcile.Result{}, errors.Wrap(err, "error reconciling routes")
	}

	tstAdminCredentials := &v1.Secret{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: tst.Spec.AdminCredentials, Namespace: tst.Namespace}, tstAdminCredentials)
	if err != nil {
		return reconcile.Result{}, errors.Wrap(err, "failed to get admin credentials")
	}

	tsClient, err := r.tsFactory.AuthenticatedClient(string(tstAdminCredentials.Data["ADMIN_URL"]), string(tstAdminCredentials.Data["ADMIN_ACCESS_TOKEN"]))
	if err != nil {
		return reconcile.Result{}, errors.Wrap(err, "failed to get authenticated client for threescale")
	}

	tst, err = r.ReconcileAuthProviders(tst, tsClient)
	if err != nil {
		return reconcile.Result{}, errors.Wrap(err, "error reconciling auth providers")
	}
	log.Info("Reconcile Authentication Providers: done")
	tst, err = r.ReconcileUsers(tst, tsClient)
	if err != nil {
		return reconcile.Result{}, errors.Wrap(err, "error reconciling users")
	}
	log.Info("Reconcile Users: done")

	return reconcile.Result{Requeue: true, RequeueAfter: resyncDuration}, r.client.Update(context.TODO(), tst)
}

func (r *ReconcileThreeScaleTenant) createRoute(name, host, port, serviceName string, owner *threescalev1alpha1.ThreeScaleTenant) (*routev1.Route, error) {
	route := &routev1.Route{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Route",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: owner.Namespace,
			Name:      name,
		},
		Spec: routev1.RouteSpec{
			Host: host,
			Port: &routev1.RoutePort{
				TargetPort: intstr.IntOrString{
					Type:   intstr.String,
					StrVal: port,
				},
			},
			TLS: &routev1.TLSConfig{
				InsecureEdgeTerminationPolicy: routev1.InsecureEdgeTerminationPolicyNone,
				Termination:                   routev1.TLSTerminationEdge,
			},
			To: routev1.RouteTargetReference{
				Kind: "Service",
				Name: serviceName,
			},
		},
	}

	if err := controllerutil.SetControllerReference(owner, route, r.scheme); err != nil {
		return nil, err
	}

	found := &routev1.Route{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: route.Name, Namespace: route.Namespace}, found)
	if err != nil && k8errors.IsNotFound(err) {
		log.Infof("Creating new route %s/%s\n", route.Namespace, route.Name)
		err = r.client.Create(context.TODO(), route)
		if err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	} else {
		route = found
	}
	return route, nil
}

func (r *ReconcileThreeScaleTenant) createOpaqueSecret(name, value, key string, owner *threescalev1alpha1.ThreeScaleTenant) (*v1.Secret, error) {
	data := map[string][]byte{key: []byte(value)}
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
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: secret.Name, Namespace: secret.Namespace}, found)
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

func (r *ReconcileThreeScaleTenant) CreateTenant(ts *threescalev1alpha1.ThreeScale, tst *threescalev1alpha1.ThreeScaleTenant) (*threescalev1alpha1.ThreeScaleTenant, error) {
	if tst.Spec.Tenant.ID == 0 {
		tsMasterCredentials := &v1.Secret{}
		err := r.client.Get(context.TODO(), types.NamespacedName{Name: ts.Spec.MasterCredentials, Namespace: ts.Namespace}, tsMasterCredentials)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get master credentials")
		}

		tsClient, err := r.tsFactory.AuthenticatedClient(string(tsMasterCredentials.Data["ADMIN_URL"]), string(tsMasterCredentials.Data["MASTER_ACCESS_TOKEN"]))
		if err != nil {
			return nil, errors.Wrap(err, "failed to get authenticated client for threescale")
		}

		signupRequest := &threescalev1alpha1.ThreeScaleSignupRequest{
			OrgName:  tst.Spec.Name,
			Email:    tst.Spec.AdminEmail,
			UserName: tst.Spec.AdminUsername,
			Password: tst.Spec.AdminPassword,
		}

		signupResponse, err := tsClient.CreateTenant(signupRequest)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create signupRequest")
		}
		tst.Spec.Tenant = *signupResponse.Signup.Account
		r.client.Update(context.TODO(), tst) //Todo

		secretName := fmt.Sprintf("%s-provider-admin-credentials", tst.Spec.Name)
		adminSecret, err := r.createOpaqueSecret(secretName, signupResponse.Signup.AccessToken.Value, "ADMIN_ACCESS_TOKEN", tst)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create admin token")
		}
		tst.Spec.AdminCredentials = adminSecret.GetName()
		r.client.Update(context.TODO(), tst) //Todo
	}
	return tst, nil
}

func (r *ReconcileThreeScaleTenant) ReconcileRoutes(tst *threescalev1alpha1.ThreeScaleTenant) (*threescalev1alpha1.ThreeScaleTenant, error) {
	adminRouteName := fmt.Sprintf("%s-provider-admin-route", tst.Spec.Name)
	adminRoute, err := r.createRoute(adminRouteName, tst.Spec.Tenant.AdminDomain, "http", "system-provider", tst)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create/find admin route")
	}

	protocol := "https"
	if adminRoute.Spec.TLS == nil {
		protocol = "http"
	}
	adminUrl := fmt.Sprintf("%v://%v", protocol, adminRoute.Spec.Host)

	adminCreds := &v1.Secret{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: tst.Spec.AdminCredentials, Namespace: tst.Namespace}, adminCreds)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get the secret for the admin credentials")
	}
	adminCreds.Data["ADMIN_URL"] = []byte(adminUrl)

	if err = r.client.Update(context.TODO(), adminCreds); err != nil {
		return nil, errors.Wrap(err, "could not update admin credentials")
	}

	routeName := fmt.Sprintf("%s-provider-developer-route", tst.Spec.Name)
	_, err = r.createRoute(routeName, tst.Spec.Tenant.Domain, "http", "system-developer", tst)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create/find developer route")
	}

	return tst, nil
}

func (r *ReconcileThreeScaleTenant) ReconcileAuthProviders(tst *threescalev1alpha1.ThreeScaleTenant, tsClient threescale.ThreeScaleInterface) (*threescalev1alpha1.ThreeScaleTenant, error) {
	apiAuthProviders, err := tsClient.ListAuthProviders()
	if err != nil {
		return nil, errors.Wrap(err, "error retrieving auth providers from threescale")
	}
	tsAuthProviders := map[string]*threescalev1alpha1.ThreeScaleAuthProvider{}
	for i := range apiAuthProviders {
		tsAuthProviders[apiAuthProviders[i].AuthProvider.Name] = apiAuthProviders[i].AuthProvider
	}

	specAuthProviders := map[string]*threescalev1alpha1.ThreeScaleAuthProvider{}
	for i := range tst.Spec.AuthProviders {
		specAuthProviders[tst.Spec.AuthProviders[i].Name] = &tst.Spec.AuthProviders[i]
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

	return tst, nil
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

func (r *ReconcileThreeScaleTenant) ReconcileUsers(tst *threescalev1alpha1.ThreeScaleTenant, tsClient threescale.ThreeScaleInterface) (*threescalev1alpha1.ThreeScaleTenant, error) {
	apiUsers, err := tsClient.ListUsers()
	if err != nil {
		return nil, errors.Wrap(err, "error retrieving apiUsers from threescale")
	}
	tsUsers := map[string]*threescalev1alpha1.ThreeScaleUser{}
	for i := range apiUsers {
		tsUsers[apiUsers[i].User.UserName] = apiUsers[i].User
	}

	specUsers := map[string]*threescalev1alpha1.ThreeScaleUser{}
	for i := range tst.Spec.Users {
		specUsers[tst.Spec.Users[i].UserName] = &tst.Spec.Users[i]
	}

	if tst.Spec.SeedUsers.Count != 0 {
		for i := 1; i <= tst.Spec.SeedUsers.Count; i++ {
			username := fmt.Sprintf(tst.Spec.SeedUsers.NameFormat, i)
			if specUsers[username] == nil {
				evalUser := &threescalev1alpha1.ThreeScaleUser{
					Email:    fmt.Sprintf(tst.Spec.SeedUsers.EmailFormat, i),
					UserName: username,
					Password: tst.Spec.SeedUsers.Password,
					Role:     tst.Spec.SeedUsers.Role,
				}
				tst.Spec.Users = append(tst.Spec.Users, *evalUser)
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

	return tst, nil
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

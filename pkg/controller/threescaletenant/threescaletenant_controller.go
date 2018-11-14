package threescaletenant

import (
	"context"
	"fmt"
	"github.com/integr8ly/3scale-operator/pkg/threescale"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"reflect"
	"time"

	threescalev1alpha1 "github.com/integr8ly/3scale-operator/pkg/apis/threescale/v1alpha1"
	k8errors "k8s.io/apimachinery/pkg/api/errors"
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

	tsClient, err := r.tsFactory.AuthenticatedClient(*tst)
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

	resyncDuration := time.Second * time.Duration(10)
	return reconcile.Result{Requeue: true, RequeueAfter: resyncDuration}, r.client.Update(context.TODO(), tst)
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

package stub

import (
	"context"
	"github.com/integr8ly/3scale-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/3scale-operator/pkg/clients/openshift"
	"github.com/integr8ly/3scale-operator/pkg/threescale"
	"github.com/operator-framework/operator-sdk/pkg/k8sclient"
	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
)

func NewHandler(k8client kubernetes.Interface, osClient *openshift.ClientFactory, tsFactory *threescale.ThreeScaleFactory) sdk.Handler {
	return &Handler{
		phaseHandler: NewPhaseHandler(k8client, osClient, tsFactory, k8sclient.GetResourceClient),
	}
}

type Handler struct {
	phaseHandler *phaseHandler
}

func (h *Handler) Handle(ctx context.Context, event sdk.Event) error {
	logrus.Debug("handling object ", event.Object.GetObjectKind().GroupVersionKind().String())

	if event.Deleted {
		return nil
	}

	switch o := event.Object.(type) {
	case *v1alpha1.ThreeScale:
		logrus.Debugf("ThreeScale: %v, Phase: %v", o.Name, o.Status.Phase)
		if event.Deleted {
			return nil
		}
		switch o.Status.Phase {
		case v1alpha1.NoPhase:
			tsState, err := h.phaseHandler.Initialise(o)
			if err != nil {
				return errors.Wrap(err, "failed to init resource")
			}
			return sdk.Update(tsState)
		case v1alpha1.PhaseProvisionCredentials:
			tsState, err := h.phaseHandler.Credentials(o)
			if err != nil {
				return errors.Wrap(err, "phase provision credentials failed")
			}
			return sdk.Update(tsState)
		case v1alpha1.PhaseReconcileThreescale:
			tsState, err := h.phaseHandler.ReconcileThreeScale(o)
			if err != nil {
				return errors.Wrap(err, "phase reconcile threescale failed")
			}
			return sdk.Update(tsState)
		}
		return nil
	}
	return nil
}

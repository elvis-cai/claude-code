package controllers

import (
	"context"
	"sync"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	alertsv1 "github.com/vibe-coding/pod-restart-slack-operator/api/v1"
	"github.com/vibe-coding/pod-restart-slack-operator/pkg/events"
)

type SlackAlertReconciler struct {
	client.Client
	Scheme *runtime.Scheme

	eventWatcher *events.PodEventWatcher
	watcherOnce  sync.Once
}

//+kubebuilder:rbac:groups=alerts.vibe-coding.com,resources=slackalerts,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=alerts.vibe-coding.com,resources=slackalerts/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=alerts.vibe-coding.com,resources=slackalerts/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=events,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch

func (r *SlackAlertReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	slackAlert := &alertsv1.SlackAlert{}
	err := r.Get(ctx, req.NamespacedName, slackAlert)
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Info("SlackAlert resource not found. Ignoring since object must be deleted")
			r.eventWatcher.RemoveAlertConfig(req.Namespace, req.Name)
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Failed to get SlackAlert")
		return ctrl.Result{}, err
	}

	r.ensureEventWatcher()

	if slackAlert.DeletionTimestamp != nil {
		r.eventWatcher.RemoveAlertConfig(slackAlert.Namespace, slackAlert.Name)
		return ctrl.Result{}, nil
	}

	r.eventWatcher.UpdateAlertConfig(slackAlert)

	logger.Info("Reconciled SlackAlert", "name", slackAlert.Name, "namespace", slackAlert.Namespace, "enabled", slackAlert.Spec.Enabled)

	return ctrl.Result{}, nil
}

func (r *SlackAlertReconciler) ensureEventWatcher() {
	r.watcherOnce.Do(func() {
		config := ctrl.GetConfigOrDie()
		k8sClient, err := kubernetes.NewForConfig(config)
		if err != nil {
			log.Log.Error(err, "Failed to create Kubernetes client")
			return
		}

		r.eventWatcher = events.NewPodEventWatcher(r.Client, k8sClient)

		go func() {
			if err := r.eventWatcher.WatchPodEvents(context.Background()); err != nil {
				log.Log.Error(err, "Event watcher stopped")
			}
		}()
	})
}

func (r *SlackAlertReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&alertsv1.SlackAlert{}).
		Complete(r)
}

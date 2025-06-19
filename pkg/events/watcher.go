package events

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	alertsv1 "github.com/vibe-coding/pod-restart-slack-operator/api/v1"
	"github.com/vibe-coding/pod-restart-slack-operator/pkg/slack"
)

type PodEventWatcher struct {
	client       client.Client
	k8sClient    kubernetes.Interface
	slackClients map[string]*slack.Client
	alertConfigs map[string]*alertsv1.SlackAlert
}

func NewPodEventWatcher(client client.Client, k8sClient kubernetes.Interface) *PodEventWatcher {
	return &PodEventWatcher{
		client:       client,
		k8sClient:    k8sClient,
		slackClients: make(map[string]*slack.Client),
		alertConfigs: make(map[string]*alertsv1.SlackAlert),
	}
}

func (w *PodEventWatcher) UpdateAlertConfig(alert *alertsv1.SlackAlert) {
	key := fmt.Sprintf("%s/%s", alert.Namespace, alert.Name)
	w.alertConfigs[key] = alert

	if alert.Spec.Enabled {
		w.slackClients[key] = slack.NewClient(alert.Spec.WebhookURL)
	} else {
		delete(w.slackClients, key)
	}
}

func (w *PodEventWatcher) RemoveAlertConfig(namespace, name string) {
	key := fmt.Sprintf("%s/%s", namespace, name)
	delete(w.alertConfigs, key)
	delete(w.slackClients, key)
}

func (w *PodEventWatcher) WatchPodEvents(ctx context.Context) error {
	logger := log.FromContext(ctx).WithName("pod-event-watcher")

	watchlist := w.k8sClient.CoreV1().Events("")

	fieldSelector := fields.AndSelectors(
		fields.OneTermEqualSelector("involvedObject.kind", "Pod"),
		fields.OneTermEqualSelector("type", "Warning"),
	)

	watcher, err := watchlist.Watch(ctx, metav1.ListOptions{
		FieldSelector: fieldSelector.String(),
	})
	if err != nil {
		return fmt.Errorf("failed to create event watcher: %w", err)
	}
	defer watcher.Stop()

	logger.Info("Started watching pod events")

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case event, ok := <-watcher.ResultChan():
			if !ok {
				logger.Info("Event watcher channel closed, restarting...")
				return w.WatchPodEvents(ctx)
			}

			k8sEvent, ok := event.Object.(*corev1.Event)
			if !ok {
				continue
			}

			if w.isPodRestartEvent(k8sEvent) {
				logger.Info("Detected pod restart event", "pod", k8sEvent.InvolvedObject.Name, "namespace", k8sEvent.InvolvedObject.Namespace, "reason", k8sEvent.Reason)
				if err := w.handlePodRestartEvent(ctx, k8sEvent); err != nil {
					logger.Error(err, "Failed to handle pod restart event", "event", k8sEvent.Name, "pod", k8sEvent.InvolvedObject.Name)
				}
			}
		}
	}
}

func (w *PodEventWatcher) isPodRestartEvent(event *corev1.Event) bool {
	restartReasons := []string{
		"Killing",
		"BackOff",
		"FailedMount",
		"Failed",
		"Unhealthy",
	}

	for _, reason := range restartReasons {
		if event.Reason == reason {
			return true
		}
	}

	return strings.Contains(strings.ToLower(event.Message), "restart") ||
		strings.Contains(strings.ToLower(event.Message), "restarting") ||
		strings.Contains(strings.ToLower(event.Message), "killed")
}

func (w *PodEventWatcher) handlePodRestartEvent(ctx context.Context, event *corev1.Event) error {
	logger := log.FromContext(ctx)

	pod := &corev1.Pod{}
	err := w.client.Get(ctx, client.ObjectKey{
		Name:      event.InvolvedObject.Name,
		Namespace: event.InvolvedObject.Namespace,
	}, pod)
	if err != nil {
		return fmt.Errorf("failed to get pod %s/%s: %w", event.InvolvedObject.Namespace, event.InvolvedObject.Name, err)
	}

	for key, alert := range w.alertConfigs {
		if !alert.Spec.Enabled {
			continue
		}

		if w.shouldSendAlert(alert, pod) {
			slackClient := w.slackClients[key]
			if slackClient == nil {
				continue
			}

			err := slackClient.SendPodRestartAlert(
				pod,
				event,
				alert.Spec.Channel,
				alert.Spec.Username,
				alert.Spec.MessageTemplate,
			)
			if err != nil {
				logger.Error(err, "Failed to send Slack alert", "alert", key, "pod", pod.Name)
				continue
			}

			logger.Info("Sent pod restart alert", "alert", key, "pod", pod.Name, "namespace", pod.Namespace)

			w.updateAlertStatus(ctx, alert, event)
		}
	}

	return nil
}

func (w *PodEventWatcher) shouldSendAlert(alert *alertsv1.SlackAlert, pod *corev1.Pod) bool {
	if alert.Spec.NamespaceSelector != nil {
		selector, err := metav1.LabelSelectorAsSelector(alert.Spec.NamespaceSelector)
		if err != nil {
			return false
		}

		namespace := &corev1.Namespace{}
		err = w.client.Get(context.Background(), client.ObjectKey{Name: pod.Namespace}, namespace)
		if err != nil {
			return false
		}

		if !selector.Matches(labels.Set(namespace.Labels)) {
			return false
		}
	}

	if alert.Spec.PodSelector != nil {
		selector, err := metav1.LabelSelectorAsSelector(alert.Spec.PodSelector)
		if err != nil {
			return false
		}

		if !selector.Matches(labels.Set(pod.Labels)) {
			return false
		}
	}

	return true
}

func (w *PodEventWatcher) updateAlertStatus(ctx context.Context, alert *alertsv1.SlackAlert, event *corev1.Event) {
	now := metav1.Now()
	alert.Status.LastEventTime = &now
	alert.Status.EventCount++

	w.client.Status().Update(ctx, alert)
}

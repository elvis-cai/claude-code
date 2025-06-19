package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type SlackAlertSpec struct {
	WebhookURL string `json:"webhookUrl"`
	Channel    string `json:"channel,omitempty"`
	Username   string `json:"username,omitempty"`

	NamespaceSelector *metav1.LabelSelector `json:"namespaceSelector,omitempty"`
	PodSelector       *metav1.LabelSelector `json:"podSelector,omitempty"`

	Enabled bool `json:"enabled"`

	MessageTemplate string `json:"messageTemplate,omitempty"`
}

type SlackAlertStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	LastEventTime *metav1.Time `json:"lastEventTime,omitempty"`

	EventCount int32 `json:"eventCount,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:scope=Namespaced

type SlackAlert struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SlackAlertSpec   `json:"spec,omitempty"`
	Status SlackAlertStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

type SlackAlertList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SlackAlert `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SlackAlert{}, &SlackAlertList{})
}

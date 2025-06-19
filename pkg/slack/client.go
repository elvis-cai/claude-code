package slack

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	corev1 "k8s.io/api/core/v1"
)

type Client struct {
	webhookURL string
	httpClient *http.Client
}

type SlackMessage struct {
	Channel     string       `json:"channel,omitempty"`
	Username    string       `json:"username,omitempty"`
	Text        string       `json:"text,omitempty"`
	Attachments []Attachment `json:"attachments,omitempty"`
}

type Attachment struct {
	Color     string  `json:"color,omitempty"`
	Title     string  `json:"title,omitempty"`
	Text      string  `json:"text,omitempty"`
	Fields    []Field `json:"fields,omitempty"`
	Timestamp int64   `json:"ts,omitempty"`
}

type Field struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short"`
}

func NewClient(webhookURL string) *Client {
	return &Client{
		webhookURL: webhookURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) SendPodRestartAlert(pod *corev1.Pod, event *corev1.Event, channel, username, template string) error {
	var message SlackMessage

	if template != "" {
		message.Text = c.formatCustomMessage(template, pod, event)
	} else {
		message = c.createDefaultMessage(pod, event)
	}

	if channel != "" {
		message.Channel = channel
	}
	if username != "" {
		message.Username = username
	}

	if err := c.sendMessage(message); err != nil {
		return fmt.Errorf("failed to send slack alert for pod %s/%s: %w", pod.Namespace, pod.Name, err)
	}

	return nil
}

func (c *Client) createDefaultMessage(pod *corev1.Pod, event *corev1.Event) SlackMessage {
	color := "warning"
	if event.Reason == "Failed" {
		color = "danger"
	}

	attachment := Attachment{
		Color:     color,
		Title:     fmt.Sprintf("Pod Restart Alert: %s", pod.Name),
		Text:      fmt.Sprintf("Pod `%s` in namespace `%s` has restarted", pod.Name, pod.Namespace),
		Timestamp: time.Now().Unix(),
		Fields: []Field{
			{
				Title: "Namespace",
				Value: pod.Namespace,
				Short: true,
			},
			{
				Title: "Pod Name",
				Value: pod.Name,
				Short: true,
			},
			{
				Title: "Reason",
				Value: event.Reason,
				Short: true,
			},
			{
				Title: "Message",
				Value: event.Message,
				Short: false,
			},
		},
	}

	if pod.Status.ContainerStatuses != nil {
		for _, cs := range pod.Status.ContainerStatuses {
			attachment.Fields = append(attachment.Fields, Field{
				Title: fmt.Sprintf("Container: %s", cs.Name),
				Value: fmt.Sprintf("Restart Count: %d", cs.RestartCount),
				Short: true,
			})
		}
	}

	return SlackMessage{
		Attachments: []Attachment{attachment},
	}
}

func (c *Client) formatCustomMessage(template string, pod *corev1.Pod, event *corev1.Event) string {
	message := template
	message = fmt.Sprintf(message, pod.Name, pod.Namespace, event.Reason, event.Message)
	return message
}

func (c *Client) sendMessage(message SlackMessage) error {
	payload, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal slack message: %w", err)
	}

	resp, err := c.httpClient.Post(c.webhookURL, "application/json", bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("failed to send slack message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("slack webhook returned non-200 status: %d", resp.StatusCode)
	}

	return nil
}

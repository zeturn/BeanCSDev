package k8s

import (
	"bytes"
	"context"
	"io"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (m *Manager) PodStatus(ctx context.Context, namespace, projectName string) ([]corev1.Pod, error) {
	if err := m.ensure(); err != nil {
		return nil, err
	}
	list, err := m.Clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{LabelSelector: "app=" + projectName + ",managed-by=beancs"})
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

func (m *Manager) Logs(ctx context.Context, namespace, projectName string, tail int64) (string, error) {
	if err := m.ensure(); err != nil {
		return "", err
	}
	pods, err := m.PodStatus(ctx, namespace, projectName)
	if err != nil {
		return "", err
	}
	if len(pods) == 0 {
		return "", nil
	}
	req := m.Clientset.CoreV1().Pods(namespace).GetLogs(pods[0].Name, &corev1.PodLogOptions{TailLines: &tail})
	stream, err := req.Stream(ctx)
	if err != nil {
		return "", err
	}
	defer stream.Close()
	var buf bytes.Buffer
	_, err = io.Copy(&buf, stream)
	return buf.String(), err
}

func (m *Manager) Nodes(ctx context.Context) ([]corev1.Node, error) {
	if err := m.ensure(); err != nil {
		return nil, err
	}
	list, err := m.Clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

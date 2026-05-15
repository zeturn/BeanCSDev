package k8s

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/zeturn/beancs-controller/internal/model"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ConnectivityCheck struct {
	Name    string
	Target  string
	Status  string
	Message string
}

func (m *Manager) WaitForProjectConnectivity(ctx context.Context, namespace, projectName string, ports model.ProjectPorts, timeout time.Duration) ([]ConnectivityCheck, error) {
	deadline := time.Now().Add(timeout)
	var last []ConnectivityCheck
	var lastErr error
	for {
		checks, err := m.ProjectConnectivity(ctx, namespace, projectName, ports)
		if err == nil {
			return checks, nil
		}
		last = checks
		lastErr = err
		if time.Now().After(deadline) {
			if len(last) > 0 {
				return last, lastErr
			}
			return nil, lastErr
		}
		select {
		case <-ctx.Done():
			return last, ctx.Err()
		case <-time.After(5 * time.Second):
		}
	}
}

func (m *Manager) ProjectConnectivity(ctx context.Context, namespace, projectName string, ports model.ProjectPorts) ([]ConnectivityCheck, error) {
	if err := m.ensure(); err != nil {
		return nil, err
	}
	port, err := firstServicePort(ports)
	if err != nil {
		return nil, err
	}
	checks := []ConnectivityCheck{}
	endpointCheck, endpointErr := m.endpointConnectivity(ctx, namespace, projectName, port.Port)
	checks = append(checks, endpointCheck)
	if endpointErr != nil {
		return checks, endpointErr
	}
	serviceTarget := fmt.Sprintf("http://%s.%s.svc.cluster.local:%d", projectName, namespace, port.Port)
	serviceCheck, serviceErr := httpConnectivity(ctx, "cluster-service", serviceTarget+"/health")
	if serviceErr != nil {
		serviceCheck, serviceErr = httpConnectivity(ctx, "cluster-service", serviceTarget+"/")
	}
	checks = append(checks, serviceCheck)
	if serviceErr != nil {
		return checks, serviceErr
	}
	return checks, nil
}

func (m *Manager) endpointConnectivity(ctx context.Context, namespace, serviceName string, port int) (ConnectivityCheck, error) {
	target := fmt.Sprintf("%s/%s:%d", namespace, serviceName, port)
	endpoints, err := m.Clientset.CoreV1().Endpoints(namespace).Get(ctx, serviceName, metav1.GetOptions{})
	if err != nil {
		return ConnectivityCheck{Name: "endpoints", Target: target, Status: "failed", Message: err.Error()}, err
	}
	addresses := 0
	for _, subset := range endpoints.Subsets {
		for _, endpointPort := range subset.Ports {
			if int(endpointPort.Port) != port {
				continue
			}
			addresses += len(subset.Addresses)
		}
	}
	if addresses == 0 {
		err := fmt.Errorf("service has no ready endpoints on port %d", port)
		return ConnectivityCheck{Name: "endpoints", Target: target, Status: "failed", Message: err.Error()}, err
	}
	return ConnectivityCheck{Name: "endpoints", Target: target, Status: "succeeded", Message: fmt.Sprintf("%d ready endpoint(s)", addresses)}, nil
}

func httpConnectivity(ctx context.Context, name, target string) (ConnectivityCheck, error) {
	reqCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, target, nil)
	if err != nil {
		return ConnectivityCheck{Name: name, Target: target, Status: "failed", Message: err.Error()}, err
	}
	req.Header.Set("User-Agent", "BeanCS connectivity check")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return ConnectivityCheck{Name: name, Target: target, Status: "failed", Message: err.Error()}, err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		err := fmt.Errorf("unexpected HTTP status %d", resp.StatusCode)
		return ConnectivityCheck{Name: name, Target: target, Status: "failed", Message: err.Error()}, err
	}
	return ConnectivityCheck{Name: name, Target: target, Status: "succeeded", Message: resp.Status}, nil
}

func firstServicePort(ports model.ProjectPorts) (model.ProjectPort, error) {
	for _, port := range ports {
		if port.Port > 0 {
			return port, nil
		}
	}
	return model.ProjectPort{}, fmt.Errorf("project has no valid service port")
}

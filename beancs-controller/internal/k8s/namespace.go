package k8s

import (
	"context"
	"fmt"
	"time"

	"github.com/zeturn/beancs-controller/internal/dto"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type NamespaceSummary struct {
	Name       string            `json:"name"`
	Status     string            `json:"status"`
	Labels     map[string]string `json:"labels,omitempty"`
	AgeSeconds int64             `json:"age_seconds"`
	CreatedAt  time.Time         `json:"created_at"`
}

type NamespaceDetail struct {
	Summary        NamespaceSummary              `json:"summary"`
	Stats          NamespaceStats                `json:"stats"`
	ResourceQuotas []ResourceQuotaSummary        `json:"resource_quotas"`
	LimitRanges    []LimitRangeSummary           `json:"limit_ranges"`
	Roles          []NamespaceRoleSummary        `json:"roles"`
	RoleBindings   []NamespaceRoleBindingSummary `json:"role_bindings"`
	Isolation      NamespaceIsolationSummary     `json:"isolation"`
	CheckedAt      time.Time                     `json:"checked_at"`
}

type NamespaceStats struct {
	Pods            int `json:"pods"`
	RunningPods     int `json:"running_pods"`
	AbnormalPods    int `json:"abnormal_pods"`
	Services        int `json:"services"`
	Deployments     int `json:"deployments"`
	Ingresses       int `json:"ingresses"`
	NetworkPolicies int `json:"network_policies"`
	Secrets         int `json:"secrets"`
	ConfigMaps      int `json:"config_maps"`
}

type ResourceQuotaSummary struct {
	Name string            `json:"name"`
	Hard map[string]string `json:"hard"`
	Used map[string]string `json:"used"`
}

type LimitRangeSummary struct {
	Name  string   `json:"name"`
	Types []string `json:"types"`
}

type NamespaceRoleSummary struct {
	Name  string `json:"name"`
	Rules int    `json:"rules"`
}

type NamespaceRoleBindingSummary struct {
	Name     string   `json:"name"`
	RoleRef  string   `json:"role_ref"`
	Subjects []string `json:"subjects"`
}

type NamespaceIsolationSummary struct {
	Enabled            bool   `json:"enabled"`
	PolicyName         string `json:"policy_name,omitempty"`
	AllowSameNamespace bool   `json:"allow_same_namespace"`
	AllowDNS           bool   `json:"allow_dns"`
}

func (m *Manager) CreateNamespace(ctx context.Context, name, projectName string) error {
	if err := m.ensure(); err != nil {
		return err
	}
	_, err := m.Clientset.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: name, Labels: Labels(projectName)},
	}, metav1.CreateOptions{})
	if apierrors.IsAlreadyExists(err) {
		return nil
	}
	return err
}

func (m *Manager) DeleteNamespace(ctx context.Context, name string) error {
	if err := m.ensure(); err != nil {
		return err
	}
	err := m.Clientset.CoreV1().Namespaces().Delete(ctx, name, metav1.DeleteOptions{})
	if apierrors.IsNotFound(err) {
		return nil
	}
	return err
}

func (m *Manager) ListNamespaces(ctx context.Context) ([]NamespaceSummary, error) {
	if err := m.ensure(); err != nil {
		return nil, err
	}
	list, err := m.Clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	out := make([]NamespaceSummary, 0, len(list.Items))
	now := time.Now()
	for _, ns := range list.Items {
		created := ns.CreationTimestamp.Time
		out = append(out, NamespaceSummary{
			Name:       ns.Name,
			Status:     string(ns.Status.Phase),
			Labels:     ns.Labels,
			AgeSeconds: int64(now.Sub(created).Seconds()),
			CreatedAt:  created,
		})
	}
	return out, nil
}

func (m *Manager) NamespaceDetail(ctx context.Context, name string) (*NamespaceDetail, error) {
	if err := m.ensure(); err != nil {
		return nil, err
	}
	ns, err := m.Clientset.CoreV1().Namespaces().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	stats, err := m.namespaceStats(ctx, name)
	if err != nil {
		return nil, err
	}
	quotas, err := m.namespaceQuotas(ctx, name)
	if err != nil {
		return nil, err
	}
	limits, err := m.namespaceLimitRanges(ctx, name)
	if err != nil {
		return nil, err
	}
	roles, roleBindings, err := m.namespacePermissions(ctx, name)
	if err != nil {
		return nil, err
	}
	isolation, err := m.namespaceIsolation(ctx, name)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	created := ns.CreationTimestamp.Time
	return &NamespaceDetail{
		Summary: NamespaceSummary{
			Name:       ns.Name,
			Status:     string(ns.Status.Phase),
			Labels:     ns.Labels,
			AgeSeconds: int64(now.Sub(created).Seconds()),
			CreatedAt:  created,
		},
		Stats:          stats,
		ResourceQuotas: quotas,
		LimitRanges:    limits,
		Roles:          roles,
		RoleBindings:   roleBindings,
		Isolation:      isolation,
		CheckedAt:      time.Now().UTC(),
	}, nil
}

func (m *Manager) UpsertResourceQuota(ctx context.Context, namespace string, req dto.ResourceQuotaRequest) error {
	if err := m.ensure(); err != nil {
		return err
	}
	hard, err := resourceListFromStrings(req.Hard)
	if err != nil {
		return err
	}
	quota := &corev1.ResourceQuota{
		ObjectMeta: metav1.ObjectMeta{Name: req.Name, Namespace: namespace, Labels: map[string]string{"managed-by": "beancs"}},
		Spec:       corev1.ResourceQuotaSpec{Hard: hard},
	}
	current, err := m.Clientset.CoreV1().ResourceQuotas(namespace).Get(ctx, req.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		_, err = m.Clientset.CoreV1().ResourceQuotas(namespace).Create(ctx, quota, metav1.CreateOptions{})
		return err
	}
	if err != nil {
		return err
	}
	current.Labels = quota.Labels
	current.Spec = quota.Spec
	_, err = m.Clientset.CoreV1().ResourceQuotas(namespace).Update(ctx, current, metav1.UpdateOptions{})
	return err
}

func (m *Manager) DeleteResourceQuota(ctx context.Context, namespace, name string) error {
	if err := m.ensure(); err != nil {
		return err
	}
	err := m.Clientset.CoreV1().ResourceQuotas(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if apierrors.IsNotFound(err) {
		return nil
	}
	return err
}

func (m *Manager) UpsertLimitRange(ctx context.Context, namespace string, req dto.LimitRangeRequest) error {
	if err := m.ensure(); err != nil {
		return err
	}
	item := corev1.LimitRangeItem{Type: corev1.LimitType(req.Type)}
	var err error
	if item.Default, err = resourceListFromStrings(req.Default); err != nil {
		return err
	}
	if item.DefaultRequest, err = resourceListFromStrings(req.DefaultRequest); err != nil {
		return err
	}
	if item.Min, err = resourceListFromStrings(req.Min); err != nil {
		return err
	}
	if item.Max, err = resourceListFromStrings(req.Max); err != nil {
		return err
	}
	limit := &corev1.LimitRange{
		ObjectMeta: metav1.ObjectMeta{Name: req.Name, Namespace: namespace, Labels: map[string]string{"managed-by": "beancs"}},
		Spec:       corev1.LimitRangeSpec{Limits: []corev1.LimitRangeItem{item}},
	}
	current, err := m.Clientset.CoreV1().LimitRanges(namespace).Get(ctx, req.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		_, err = m.Clientset.CoreV1().LimitRanges(namespace).Create(ctx, limit, metav1.CreateOptions{})
		return err
	}
	if err != nil {
		return err
	}
	current.Labels = limit.Labels
	current.Spec = limit.Spec
	_, err = m.Clientset.CoreV1().LimitRanges(namespace).Update(ctx, current, metav1.UpdateOptions{})
	return err
}

func (m *Manager) DeleteLimitRange(ctx context.Context, namespace, name string) error {
	if err := m.ensure(); err != nil {
		return err
	}
	err := m.Clientset.CoreV1().LimitRanges(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if apierrors.IsNotFound(err) {
		return nil
	}
	return err
}

func (m *Manager) UpsertNamespacePermission(ctx context.Context, namespace string, req dto.NamespacePermissionRequest) error {
	if err := m.ensure(); err != nil {
		return err
	}
	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{Name: req.Name, Namespace: namespace, Labels: map[string]string{"managed-by": "beancs"}},
		Rules: []rbacv1.PolicyRule{{
			APIGroups: req.APIGroups,
			Resources: req.Resources,
			Verbs:     req.Verbs,
		}},
	}
	if err := m.upsertRole(ctx, role); err != nil {
		return err
	}
	binding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: req.Name, Namespace: namespace, Labels: map[string]string{"managed-by": "beancs"}},
		RoleRef:    rbacv1.RoleRef{APIGroup: "rbac.authorization.k8s.io", Kind: "Role", Name: req.Name},
		Subjects:   permissionSubjects(namespace, req.Subjects),
	}
	return m.upsertRoleBinding(ctx, binding)
}

func (m *Manager) DeleteNamespacePermission(ctx context.Context, namespace, name string) error {
	if err := m.ensure(); err != nil {
		return err
	}
	err := m.Clientset.RbacV1().RoleBindings(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	err = m.Clientset.RbacV1().Roles(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if apierrors.IsNotFound(err) {
		return nil
	}
	return err
}

func (m *Manager) SetNamespaceIsolation(ctx context.Context, namespace string, req dto.NamespaceIsolationRequest) error {
	if !req.Enabled {
		return m.DeleteNetworkPolicy(ctx, namespace, "beancs-namespace-isolation")
	}
	return m.UpsertNetworkPolicy(ctx, dto.UpsertNetworkPolicyRequest{
		Namespace:          namespace,
		Name:               "beancs-namespace-isolation",
		PodSelector:        map[string]string{},
		PolicyTypes:        []string{"Ingress", "Egress"},
		AllowSameNamespace: req.AllowSameNamespace,
		AllowDNS:           req.AllowDNS,
		Labels:             map[string]string{"managed-by": "beancs", "beancs.io/isolation": "namespace"},
	})
}

func (m *Manager) namespaceStats(ctx context.Context, namespace string) (NamespaceStats, error) {
	var out NamespaceStats
	pods, err := m.Clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return out, err
	}
	out.Pods = len(pods.Items)
	for _, pod := range pods.Items {
		if pod.Status.Phase == corev1.PodRunning {
			out.RunningPods++
		}
		if podAbnormal(pod) {
			out.AbnormalPods++
		}
	}
	services, err := m.Clientset.CoreV1().Services(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return out, err
	}
	out.Services = len(services.Items)
	deployments, err := m.Clientset.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return out, err
	}
	out.Deployments = len(deployments.Items)
	ingresses, err := m.Clientset.NetworkingV1().Ingresses(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return out, err
	}
	out.Ingresses = len(ingresses.Items)
	policies, err := m.Clientset.NetworkingV1().NetworkPolicies(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return out, err
	}
	out.NetworkPolicies = len(policies.Items)
	secrets, err := m.Clientset.CoreV1().Secrets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return out, err
	}
	out.Secrets = len(secrets.Items)
	configMaps, err := m.Clientset.CoreV1().ConfigMaps(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return out, err
	}
	out.ConfigMaps = len(configMaps.Items)
	return out, nil
}

func (m *Manager) namespaceQuotas(ctx context.Context, namespace string) ([]ResourceQuotaSummary, error) {
	list, err := m.Clientset.CoreV1().ResourceQuotas(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	out := make([]ResourceQuotaSummary, 0, len(list.Items))
	for _, item := range list.Items {
		out = append(out, ResourceQuotaSummary{Name: item.Name, Hard: resourceListToStrings(item.Spec.Hard), Used: resourceListToStrings(item.Status.Used)})
	}
	return out, nil
}

func (m *Manager) namespaceLimitRanges(ctx context.Context, namespace string) ([]LimitRangeSummary, error) {
	list, err := m.Clientset.CoreV1().LimitRanges(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	out := make([]LimitRangeSummary, 0, len(list.Items))
	for _, item := range list.Items {
		types := make([]string, 0, len(item.Spec.Limits))
		for _, limit := range item.Spec.Limits {
			types = append(types, string(limit.Type))
		}
		out = append(out, LimitRangeSummary{Name: item.Name, Types: types})
	}
	return out, nil
}

func (m *Manager) namespacePermissions(ctx context.Context, namespace string) ([]NamespaceRoleSummary, []NamespaceRoleBindingSummary, error) {
	roles, err := m.Clientset.RbacV1().Roles(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, nil, err
	}
	roleOut := make([]NamespaceRoleSummary, 0, len(roles.Items))
	for _, role := range roles.Items {
		roleOut = append(roleOut, NamespaceRoleSummary{Name: role.Name, Rules: len(role.Rules)})
	}
	bindings, err := m.Clientset.RbacV1().RoleBindings(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, nil, err
	}
	bindingOut := make([]NamespaceRoleBindingSummary, 0, len(bindings.Items))
	for _, binding := range bindings.Items {
		bindingOut = append(bindingOut, NamespaceRoleBindingSummary{
			Name:     binding.Name,
			RoleRef:  binding.RoleRef.Kind + "/" + binding.RoleRef.Name,
			Subjects: roleBindingSubjects(binding.Subjects),
		})
	}
	return roleOut, bindingOut, nil
}

func (m *Manager) namespaceIsolation(ctx context.Context, namespace string) (NamespaceIsolationSummary, error) {
	policy, err := m.Clientset.NetworkingV1().NetworkPolicies(namespace).Get(ctx, "beancs-namespace-isolation", metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return NamespaceIsolationSummary{}, nil
	}
	if err != nil {
		return NamespaceIsolationSummary{}, err
	}
	return NamespaceIsolationSummary{
		Enabled:            true,
		PolicyName:         policy.Name,
		AllowSameNamespace: len(policy.Spec.Ingress) > 0,
		AllowDNS:           len(policy.Spec.Egress) > 0,
	}, nil
}

func (m *Manager) upsertRole(ctx context.Context, role *rbacv1.Role) error {
	current, err := m.Clientset.RbacV1().Roles(role.Namespace).Get(ctx, role.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		_, err = m.Clientset.RbacV1().Roles(role.Namespace).Create(ctx, role, metav1.CreateOptions{})
		return err
	}
	if err != nil {
		return err
	}
	current.Labels = role.Labels
	current.Rules = role.Rules
	_, err = m.Clientset.RbacV1().Roles(role.Namespace).Update(ctx, current, metav1.UpdateOptions{})
	return err
}

func (m *Manager) upsertRoleBinding(ctx context.Context, binding *rbacv1.RoleBinding) error {
	current, err := m.Clientset.RbacV1().RoleBindings(binding.Namespace).Get(ctx, binding.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		_, err = m.Clientset.RbacV1().RoleBindings(binding.Namespace).Create(ctx, binding, metav1.CreateOptions{})
		return err
	}
	if err != nil {
		return err
	}
	current.Labels = binding.Labels
	current.RoleRef = binding.RoleRef
	current.Subjects = binding.Subjects
	_, err = m.Clientset.RbacV1().RoleBindings(binding.Namespace).Update(ctx, current, metav1.UpdateOptions{})
	return err
}

func resourceListFromStrings(values map[string]string) (corev1.ResourceList, error) {
	out := corev1.ResourceList{}
	for key, value := range values {
		quantity, err := resource.ParseQuantity(value)
		if err != nil {
			return nil, fmt.Errorf("parse resource %s=%s: %w", key, value, err)
		}
		out[corev1.ResourceName(key)] = quantity
	}
	return out, nil
}

func resourceListToStrings(values corev1.ResourceList) map[string]string {
	out := map[string]string{}
	for key, value := range values {
		out[string(key)] = value.String()
	}
	return out
}

func permissionSubjects(namespace string, subjects []dto.NamespacePermissionSubject) []rbacv1.Subject {
	out := make([]rbacv1.Subject, 0, len(subjects))
	for _, subject := range subjects {
		subjectNamespace := subject.Namespace
		if subject.Kind == "ServiceAccount" && subjectNamespace == "" {
			subjectNamespace = namespace
		}
		out = append(out, rbacv1.Subject{Kind: subject.Kind, Name: subject.Name, Namespace: subjectNamespace})
	}
	return out
}

func roleBindingSubjects(subjects []rbacv1.Subject) []string {
	out := make([]string, 0, len(subjects))
	for _, subject := range subjects {
		value := subject.Kind + "/" + subject.Name
		if subject.Namespace != "" {
			value += "@" + subject.Namespace
		}
		out = append(out, value)
	}
	return out
}

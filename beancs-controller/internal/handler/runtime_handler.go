package handler

import (
	"bufio"
	"strconv"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/zeturn/beancs-controller/internal/dto"
	"github.com/zeturn/beancs-controller/internal/k8s"
	"github.com/zeturn/beancs-controller/internal/middleware"
	"gorm.io/gorm"
)

type RuntimeHandler struct {
	Base
	db  *gorm.DB
	k8s *k8s.Manager
}

func NewRuntimeHandler(db *gorm.DB, k8sManager *k8s.Manager, v *validator.Validate) *RuntimeHandler {
	return &RuntimeHandler{Base: NewBase(v), db: db, k8s: k8sManager}
}

func (h *RuntimeHandler) Register(r fiber.Router) {
	r.Get("/runtime/dashboard", h.dashboard)
	r.Get("/runtime/network/overview", h.networkOverview)
	r.Get("/runtime/overview", h.overview)
	r.Get("/runtime/nodes/join-command", h.nodeJoinCommand)
	r.Get("/runtime/nodes/:name", h.nodeDetail)
	r.Get("/runtime/nodes/:name/health", h.nodeHealth)
	r.Patch("/runtime/nodes/:name/labels", h.patchNodeLabels)
	r.Put("/runtime/nodes/:name/taints", h.updateNodeTaints)
	r.Post("/runtime/nodes/:name/cordon", h.cordonNode)
	r.Post("/runtime/nodes/:name/uncordon", h.uncordonNode)
	r.Post("/runtime/nodes/:name/drain", h.drainNode)
	r.Delete("/runtime/nodes/:name", h.deleteNode)
	r.Get("/runtime/namespaces", h.namespaces)
	r.Post("/runtime/namespaces", h.createNamespace)
	r.Get("/runtime/namespaces/:name", h.namespaceDetail)
	r.Patch("/runtime/namespaces/:name", h.patchNamespace)
	r.Put("/runtime/namespaces/:name/resource-quotas", h.upsertResourceQuota)
	r.Delete("/runtime/namespaces/:name/resource-quotas/:quota", h.deleteResourceQuota)
	r.Put("/runtime/namespaces/:name/limit-ranges", h.upsertLimitRange)
	r.Delete("/runtime/namespaces/:name/limit-ranges/:limit", h.deleteLimitRange)
	r.Put("/runtime/namespaces/:name/permissions", h.upsertNamespacePermission)
	r.Delete("/runtime/namespaces/:name/permissions/:permission", h.deleteNamespacePermission)
	r.Put("/runtime/namespaces/:name/isolation", h.setNamespaceIsolation)
	r.Delete("/runtime/namespaces/:name", h.deleteNamespace)
	r.Get("/runtime/pods/:namespace/:name/logs", h.podLogs)
	r.Delete("/runtime/pods/:namespace/:name", h.deletePod)
	r.Post("/runtime/services", h.createService)
	r.Put("/runtime/services/:namespace/:name", h.updateService)
	r.Delete("/runtime/services/:namespace/:name", h.deleteService)
	r.Post("/runtime/ingresses", h.createIngress)
	r.Put("/runtime/ingresses/:namespace/:name", h.updateIngress)
	r.Delete("/runtime/ingresses/:namespace/:name", h.deleteIngress)
	r.Post("/runtime/network-policies", h.createNetworkPolicy)
	r.Put("/runtime/network-policies/:namespace/:name", h.updateNetworkPolicy)
	r.Delete("/runtime/network-policies/:namespace/:name", h.deleteNetworkPolicy)
	r.Get("/projects/:id/status", middleware.ProjectAccess(h.db), h.status)
	r.Get("/projects/:id/logs", middleware.ProjectAccess(h.db), h.logs)
	r.Post("/projects/:id/restart", middleware.ProjectOwner(h.db), h.restart)
	r.Post("/projects/:id/scale", middleware.ProjectOwner(h.db), h.scale)
}

func (h *RuntimeHandler) createNamespace(c *fiber.Ctx) error {
	var req dto.CreateNamespaceRequest
	if err := h.parseAndValidate(c, &req); err != nil {
		return err
	}
	if err := h.k8s.CreateNamespaceWithLabels(c.UserContext(), req.Name, req.Labels); err != nil {
		return fail(c, 400, err)
	}
	return c.Status(201).JSON(fiber.Map{"status": "ok"})
}

func (h *RuntimeHandler) patchNamespace(c *fiber.Ctx) error {
	var req dto.RuntimeLabelPatchRequest
	if err := h.parseAndValidate(c, &req); err != nil {
		return err
	}
	if err := h.k8s.PatchNamespaceLabels(c.UserContext(), c.Params("name"), req.Labels); err != nil {
		return fail(c, 400, err)
	}
	return c.JSON(fiber.Map{"status": "ok"})
}

func (h *RuntimeHandler) deleteNamespace(c *fiber.Ctx) error {
	if err := h.k8s.DeleteNamespace(c.UserContext(), c.Params("name")); err != nil {
		return fail(c, 400, err)
	}
	return c.SendStatus(204)
}

func (h *RuntimeHandler) namespaceDetail(c *fiber.Ctx) error {
	out, err := h.k8s.NamespaceDetail(c.UserContext(), c.Params("name"))
	if err != nil {
		return fail(c, 400, err)
	}
	return c.JSON(fiber.Map{"data": out})
}

func (h *RuntimeHandler) upsertResourceQuota(c *fiber.Ctx) error {
	var req dto.ResourceQuotaRequest
	if err := h.parseAndValidate(c, &req); err != nil {
		return err
	}
	if err := h.k8s.UpsertResourceQuota(c.UserContext(), c.Params("name"), req); err != nil {
		return fail(c, 400, err)
	}
	return c.JSON(fiber.Map{"status": "ok"})
}

func (h *RuntimeHandler) deleteResourceQuota(c *fiber.Ctx) error {
	if err := h.k8s.DeleteResourceQuota(c.UserContext(), c.Params("name"), c.Params("quota")); err != nil {
		return fail(c, 400, err)
	}
	return c.SendStatus(204)
}

func (h *RuntimeHandler) upsertLimitRange(c *fiber.Ctx) error {
	var req dto.LimitRangeRequest
	if err := h.parseAndValidate(c, &req); err != nil {
		return err
	}
	if err := h.k8s.UpsertLimitRange(c.UserContext(), c.Params("name"), req); err != nil {
		return fail(c, 400, err)
	}
	return c.JSON(fiber.Map{"status": "ok"})
}

func (h *RuntimeHandler) deleteLimitRange(c *fiber.Ctx) error {
	if err := h.k8s.DeleteLimitRange(c.UserContext(), c.Params("name"), c.Params("limit")); err != nil {
		return fail(c, 400, err)
	}
	return c.SendStatus(204)
}

func (h *RuntimeHandler) upsertNamespacePermission(c *fiber.Ctx) error {
	var req dto.NamespacePermissionRequest
	if err := h.parseAndValidate(c, &req); err != nil {
		return err
	}
	if err := h.k8s.UpsertNamespacePermission(c.UserContext(), c.Params("name"), req); err != nil {
		return fail(c, 400, err)
	}
	return c.JSON(fiber.Map{"status": "ok"})
}

func (h *RuntimeHandler) deleteNamespacePermission(c *fiber.Ctx) error {
	if err := h.k8s.DeleteNamespacePermission(c.UserContext(), c.Params("name"), c.Params("permission")); err != nil {
		return fail(c, 400, err)
	}
	return c.SendStatus(204)
}

func (h *RuntimeHandler) setNamespaceIsolation(c *fiber.Ctx) error {
	var req dto.NamespaceIsolationRequest
	if err := h.parseAndValidate(c, &req); err != nil {
		return err
	}
	if err := h.k8s.SetNamespaceIsolation(c.UserContext(), c.Params("name"), req); err != nil {
		return fail(c, 400, err)
	}
	return c.JSON(fiber.Map{"status": "ok"})
}

func (h *RuntimeHandler) podLogs(c *fiber.Ctx) error {
	tail, _ := strconv.ParseInt(c.Query("tail", "160"), 10, 64)
	if shouldFollowLogs(c) {
		targets, err := h.k8s.PodLogTargets(c.UserContext(), c.Params("namespace"), c.Params("name"), c.Query("container"))
		if err != nil {
			return fail(c, 400, err)
		}
		return h.streamLogs(c, targets, tail)
	}
	out, err := h.k8s.PodLogs(c.UserContext(), c.Params("namespace"), c.Params("name"), tail, c.Query("container"))
	if err != nil {
		return fail(c, 400, err)
	}
	return c.JSON(fiber.Map{"logs": out})
}

func (h *RuntimeHandler) deletePod(c *fiber.Ctx) error {
	if err := h.k8s.DeletePod(c.UserContext(), c.Params("namespace"), c.Params("name")); err != nil {
		return fail(c, 400, err)
	}
	return c.SendStatus(204)
}

func (h *RuntimeHandler) createService(c *fiber.Ctx) error {
	var req dto.CreateServiceRequest
	if err := h.parseAndValidate(c, &req); err != nil {
		return err
	}
	if err := h.k8s.UpsertService(c.UserContext(), req); err != nil {
		return fail(c, 400, err)
	}
	return c.Status(201).JSON(fiber.Map{"status": "ok"})
}

func (h *RuntimeHandler) updateService(c *fiber.Ctx) error {
	var req dto.UpdateServiceRequest
	if err := h.parseAndValidate(c, &req); err != nil {
		return err
	}
	createReq := dto.CreateServiceRequest{
		Namespace:             c.Params("namespace"),
		Name:                  c.Params("name"),
		Type:                  req.Type,
		Selector:              req.Selector,
		Ports:                 req.Ports,
		Labels:                req.Labels,
		LoadBalancerIP:        req.LoadBalancerIP,
		ExternalIPs:           req.ExternalIPs,
		ExternalTrafficPolicy: req.ExternalTrafficPolicy,
	}
	if err := h.k8s.UpsertService(c.UserContext(), createReq); err != nil {
		return fail(c, 400, err)
	}
	return c.JSON(fiber.Map{"status": "ok"})
}

func (h *RuntimeHandler) deleteService(c *fiber.Ctx) error {
	if err := h.k8s.DeleteService(c.UserContext(), c.Params("namespace"), c.Params("name")); err != nil {
		return fail(c, 400, err)
	}
	return c.SendStatus(204)
}

func (h *RuntimeHandler) createIngress(c *fiber.Ctx) error {
	var req dto.CreateIngressRequest
	if err := h.parseAndValidate(c, &req); err != nil {
		return err
	}
	if err := h.k8s.UpsertIngress(c.UserContext(), req); err != nil {
		return fail(c, 400, err)
	}
	return c.Status(201).JSON(fiber.Map{"status": "ok"})
}

func (h *RuntimeHandler) updateIngress(c *fiber.Ctx) error {
	var req dto.UpdateIngressRequest
	if err := h.parseAndValidate(c, &req); err != nil {
		return err
	}
	createReq := dto.CreateIngressRequest{
		Namespace:     c.Params("namespace"),
		Name:          c.Params("name"),
		ClassName:     req.ClassName,
		Host:          req.Host,
		Path:          req.Path,
		ServiceName:   req.ServiceName,
		ServicePort:   req.ServicePort,
		TLSSecretName: req.TLSSecretName,
		Annotations:   req.Annotations,
		Labels:        req.Labels,
	}
	if err := h.k8s.UpsertIngress(c.UserContext(), createReq); err != nil {
		return fail(c, 400, err)
	}
	return c.JSON(fiber.Map{"status": "ok"})
}

func (h *RuntimeHandler) deleteIngress(c *fiber.Ctx) error {
	if err := h.k8s.DeleteIngress(c.UserContext(), c.Params("namespace"), c.Params("name")); err != nil {
		return fail(c, 400, err)
	}
	return c.SendStatus(204)
}

func (h *RuntimeHandler) createNetworkPolicy(c *fiber.Ctx) error {
	var req dto.UpsertNetworkPolicyRequest
	if err := h.parseAndValidate(c, &req); err != nil {
		return err
	}
	if err := h.k8s.UpsertNetworkPolicy(c.UserContext(), req); err != nil {
		return fail(c, 400, err)
	}
	return c.Status(201).JSON(fiber.Map{"status": "ok"})
}

func (h *RuntimeHandler) updateNetworkPolicy(c *fiber.Ctx) error {
	var req dto.UpdateNetworkPolicyRequest
	if err := h.parseAndValidate(c, &req); err != nil {
		return err
	}
	createReq := dto.UpsertNetworkPolicyRequest{
		Namespace:          c.Params("namespace"),
		Name:               c.Params("name"),
		PodSelector:        req.PodSelector,
		PolicyTypes:        req.PolicyTypes,
		AllowSameNamespace: req.AllowSameNamespace,
		AllowDNS:           req.AllowDNS,
		Labels:             req.Labels,
	}
	if err := h.k8s.UpsertNetworkPolicy(c.UserContext(), createReq); err != nil {
		return fail(c, 400, err)
	}
	return c.JSON(fiber.Map{"status": "ok"})
}

func (h *RuntimeHandler) deleteNetworkPolicy(c *fiber.Ctx) error {
	if err := h.k8s.DeleteNetworkPolicy(c.UserContext(), c.Params("namespace"), c.Params("name")); err != nil {
		return fail(c, 400, err)
	}
	return c.SendStatus(204)
}

func (h *RuntimeHandler) namespaces(c *fiber.Ctx) error {
	out, err := h.k8s.ListNamespaces(c.UserContext())
	if err != nil {
		return fail(c, 400, err)
	}
	return c.JSON(fiber.Map{"data": out})
}

func (h *RuntimeHandler) overview(c *fiber.Ctx) error {
	out, err := h.k8s.RuntimeOverview(c.UserContext())
	if err != nil {
		return fail(c, 400, err)
	}
	return c.JSON(fiber.Map{"data": out})
}

func (h *RuntimeHandler) dashboard(c *fiber.Ctx) error {
	out, err := h.k8s.ClusterDashboard(c.UserContext())
	if err != nil {
		return fail(c, 400, err)
	}
	return c.JSON(fiber.Map{"data": out})
}

func (h *RuntimeHandler) networkOverview(c *fiber.Ctx) error {
	out, err := h.k8s.NetworkOverview(c.UserContext())
	if err != nil {
		return fail(c, 400, err)
	}
	return c.JSON(fiber.Map{"data": out})
}

func (h *RuntimeHandler) nodeDetail(c *fiber.Ctx) error {
	out, err := h.k8s.NodeDetail(c.UserContext(), c.Params("name"))
	if err != nil {
		return fail(c, 400, err)
	}
	return c.JSON(fiber.Map{"data": out})
}

func (h *RuntimeHandler) patchNodeLabels(c *fiber.Ctx) error {
	var req dto.RuntimeLabelPatchRequest
	if err := h.parseAndValidate(c, &req); err != nil {
		return err
	}
	if err := h.k8s.PatchNodeLabels(c.UserContext(), c.Params("name"), req.Labels); err != nil {
		return fail(c, 400, err)
	}
	return c.JSON(fiber.Map{"status": "ok"})
}

func (h *RuntimeHandler) updateNodeTaints(c *fiber.Ctx) error {
	var req dto.RuntimeTaintRequest
	if err := h.parseAndValidate(c, &req); err != nil {
		return err
	}
	if err := h.k8s.UpdateNodeTaints(c.UserContext(), c.Params("name"), req); err != nil {
		return fail(c, 400, err)
	}
	return c.JSON(fiber.Map{"status": "ok"})
}

func (h *RuntimeHandler) cordonNode(c *fiber.Ctx) error {
	if err := h.k8s.SetNodeSchedulable(c.UserContext(), c.Params("name"), false); err != nil {
		return fail(c, 400, err)
	}
	return c.JSON(fiber.Map{"status": "ok"})
}

func (h *RuntimeHandler) uncordonNode(c *fiber.Ctx) error {
	if err := h.k8s.SetNodeSchedulable(c.UserContext(), c.Params("name"), true); err != nil {
		return fail(c, 400, err)
	}
	return c.JSON(fiber.Map{"status": "ok"})
}

func (h *RuntimeHandler) drainNode(c *fiber.Ctx) error {
	var req dto.DrainNodeRequest
	if err := h.parseAndValidate(c, &req); err != nil {
		return err
	}
	out, err := h.k8s.DrainNode(c.UserContext(), c.Params("name"), req)
	if err != nil {
		return fail(c, 400, err)
	}
	return c.JSON(fiber.Map{"data": out})
}

func (h *RuntimeHandler) deleteNode(c *fiber.Ctx) error {
	if err := h.k8s.DeleteNode(c.UserContext(), c.Params("name")); err != nil {
		return fail(c, 400, err)
	}
	return c.SendStatus(204)
}

func (h *RuntimeHandler) nodeHealth(c *fiber.Ctx) error {
	out, err := h.k8s.NodeHealth(c.UserContext(), c.Params("name"))
	if err != nil {
		return fail(c, 400, err)
	}
	return c.JSON(fiber.Map{"data": out})
}

func (h *RuntimeHandler) nodeJoinCommand(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"data": h.k8s.K3sJoinCommand(c.Query("role", "agent"))})
}

func (h *RuntimeHandler) status(c *fiber.Ctx) error {
	p := projectFromCtx(c)
	out, err := h.k8s.ProjectRuntimeStatus(c.UserContext(), p.Namespace, p.Name)
	if err != nil {
		return fail(c, 400, err)
	}
	return c.JSON(out)
}

func (h *RuntimeHandler) logs(c *fiber.Ctx) error {
	p := projectFromCtx(c)
	tail, _ := strconv.ParseInt(c.Query("tail", "100"), 10, 64)
	if shouldFollowLogs(c) {
		targets, err := h.k8s.ProjectLogTargets(c.UserContext(), p.Namespace, p.Name, c.Query("container"))
		if err != nil {
			return fail(c, 400, err)
		}
		return h.streamLogs(c, targets, tail)
	}
	out, err := h.k8s.Logs(c.UserContext(), p.Namespace, p.Name, tail, c.Query("container"))
	if err != nil {
		return fail(c, 400, err)
	}
	return c.JSON(fiber.Map{"logs": out})
}

func (h *RuntimeHandler) streamLogs(c *fiber.Ctx, targets []k8s.LogTarget, tail int64) error {
	ctx := c.UserContext()
	c.Set(fiber.HeaderContentType, "text/plain; charset=utf-8")
	c.Set(fiber.HeaderCacheControl, "no-cache")
	c.Set("X-Accel-Buffering", "no")
	c.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
		h.k8s.StreamLogs(ctx, targets, tail, true, w)
	})
	return nil
}

func shouldFollowLogs(c *fiber.Ctx) bool {
	value := strings.ToLower(strings.TrimSpace(c.Query("follow")))
	return value == "1" || value == "true" || value == "yes"
}

func (h *RuntimeHandler) restart(c *fiber.Ctx) error {
	p := projectFromCtx(c)
	if err := h.k8s.RestartDeployment(c.UserContext(), p.Namespace, p.Name); err != nil {
		return fail(c, 400, err)
	}
	return c.JSON(fiber.Map{"status": "ok"})
}

func (h *RuntimeHandler) scale(c *fiber.Ctx) error {
	p := projectFromCtx(c)
	var req dto.ScaleProjectRequest
	if err := h.parseAndValidate(c, &req); err != nil {
		return err
	}
	if err := h.k8s.ScaleDeployment(c.UserContext(), p.Namespace, p.Name, int32(req.Replicas)); err != nil {
		return fail(c, 400, err)
	}
	return c.JSON(fiber.Map{"status": "ok"})
}

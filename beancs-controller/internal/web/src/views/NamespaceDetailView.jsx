import React, { useEffect, useMemo, useRef, useState } from "react";
import * as LucideIcons from "lucide-react";
import { formatTime, formatKeyValues } from "../utils/index";
import {
  MetricCard,
  SimpleTable,
  Textarea,
  Button,
  Input,
  Select,
  Checkbox,
} from "../components/index";
import { t } from "../i18n/index";
import {
  Activity,
  AlertTriangle,
  Bell,
  Boxes,
  Box,
  CheckCircle2,
  ChevronDown,
  ChevronRight,
  Cloud,
  Coffee,
  Code2,
  Container,
  Cpu,
  Database,
  Edit3,
  FileText,
  GitBranch,
  Github,
  Globe2,
  HardDrive,
  Image as ImageIcon,
  KeyRound,
  Layers3,
  LayoutDashboard,
  LineChart,
  ListRestart,
  LoaderCircle,
  Lock,
  Menu,
  MemoryStick,
  MoreHorizontal,
  Network,
  Package,
  Play,
  Plus,
  RefreshCw,
  RotateCcw,
  Rocket,
  ScrollText,
  Search,
  Server,
  Settings,
  Shield,
  ShieldCheck,
  Trash2,
  Upload,
  X,
} from "lucide-react";
export default function NamespaceDetailView({
  detail,
  onPatchNamespace,
  onSaveResourceQuota,
  onDeleteResourceQuota,
  onSaveLimitRange,
  onDeleteLimitRange,
  onSavePermission,
  onDeletePermission,
  onSaveIsolation,
}) {
  const row = detail.row || {};
  const summary = row.summary || row;
  const namespace = summary.name || row.name;
  const stats = row.stats || {};
  const isolation = row.isolation || {};
  return (
    <div className="namespace-detail">
      {detail.loading && <p className="muted">{t("Loading namespace detail...")}</p>}
      {detail.error && <p className="error-inline">{detail.error}</p>}
      <div className="dashboard-kpis">
        <MetricCard
          icon={Boxes}
          label={t("Pods")}
          value={stats.pods || 0}
          detail={t("{running} running · {abnormal} abnormal", {
            running: stats.running_pods || 0,
            abnormal: stats.abnormal_pods || 0,
          })}
        />
        <MetricCard
          icon={Database}
          label={t("Services")}
          value={stats.services || 0}
          detail={t("{count} deployments", { count: stats.deployments || 0 })}
        />
        <MetricCard
          icon={Network}
          label={t("Ingresses")}
          value={stats.ingresses || 0}
          detail={t("{count} policies", { count: stats.network_policies || 0 })}
        />
        <MetricCard
          icon={KeyRound}
          label={t("Secrets")}
          value={stats.secrets || 0}
          detail={t("{count} configmaps", { count: stats.config_maps || 0 })}
        />
        <MetricCard
          icon={Shield}
          label={t("Isolation")}
          value={isolation.enabled ? t("On") : t("Off")}
          detail={isolation.policy_name || t("No default isolation")}
          tone={isolation.enabled ? "good" : "warning"}
        />
        <MetricCard
          icon={ListRestart}
          label={t("Checked")}
          value={formatTime(row.checked_at)}
          detail={summary.status || "-"}
        />
      </div>

      <section className="node-section">
        <h3>{t("Namespace labels")}</h3>
        <form
          className="form-grid node-edit-form"
          onSubmit={(event) => {
            event.preventDefault();
            onPatchNamespace(namespace, event.currentTarget.labels.value);
          }}
        >
          <Textarea
            name="labels"
            defaultValue={formatKeyValues(summary.labels)}
          />
          <Button variant="primary">{t("Save labels")}</Button>
        </form>
      </section>

      <section className="node-section">
        <h3>{t("ResourceQuota")}</h3>
        <form
          className="form-grid quota-form"
          onSubmit={(event) => onSaveResourceQuota(namespace, event)}
        >
          <Input
            name="name"
            placeholder="quota name"
            defaultValue="default-quota"
            required
          />
          <Input
            name="hard"
            placeholder="requests.cpu=4,requests.memory=8Gi,limits.cpu=8,pods=40"
            required
          />
          <Button variant="primary">{t("Save quota")}</Button>
        </form>
        <SimpleTable
          rows={row.resource_quotas || []}
          columns={[t("name"), t("hard"), t("used")]}
          actions={(quota) => (
            <Button
              onClick={() => onDeleteResourceQuota(namespace, quota.name)}
              variant="danger"
            >
              <Trash2 size={15} />
            </Button>
          )}
          compact
        />
      </section>

      <section className="node-section">
        <h3>{t("LimitRange")}</h3>
        <form
          className="form-grid limit-form"
          onSubmit={(event) => onSaveLimitRange(namespace, event)}
        >
          <Input
            name="name"
            placeholder="limit range name"
            defaultValue="default-limits"
            required
          />
          <Select name="type" defaultValue="Container">
            <option>Container</option>
            <option>Pod</option>
            <option>PersistentVolumeClaim</option>
          </Select>
          <Input
            name="default_request"
            placeholder="default request: cpu=100m,memory=128Mi"
          />
          <Input
            name="default"
            placeholder="default limit: cpu=500m,memory=512Mi"
          />
          <Input name="min" placeholder="min: cpu=50m,memory=64Mi" />
          <Input name="max" placeholder="max: cpu=2,memory=2Gi" />
          <Button variant="primary">{t("Save limits")}</Button>
        </form>
        <SimpleTable
          rows={row.limit_ranges || []}
          columns={[t("name"), t("types")]}
          actions={(limit) => (
            <Button
              onClick={() => onDeleteLimitRange(namespace, limit.name)}
              variant="danger"
            >
              <Trash2 size={15} />
            </Button>
          )}
          compact
        />
      </section>

      <section className="node-section">
        <h3>{t("Namespace permissions")}</h3>
        <form
          className="form-grid permission-form"
          onSubmit={(event) => onSavePermission(namespace, event)}
        >
          <Input
            name="name"
            placeholder="permission name"
            defaultValue="namespace-admin"
            required
          />
          <Input
            name="subjects"
            placeholder="subjects: User:alice,Group:platform,ServiceAccount:builder"
            required
          />
          <Input
            name="verbs"
            placeholder="verbs: get,list,watch,create,update,delete"
            defaultValue="get,list,watch"
            required
          />
          <Input
            name="resources"
            placeholder="resources: pods,services,deployments"
            defaultValue="pods,services"
            required
          />
          <Input
            name="api_groups"
            placeholder="api groups: ,apps,networking.k8s.io"
          />
          <Button variant="primary">{t("Save permission")}</Button>
        </form>
        <SimpleTable
          rows={row.role_bindings || []}
          columns={[t("name"), t("role_ref"), t("subjects")]}
          actions={(binding) => (
            <Button
              onClick={() => onDeletePermission(namespace, binding.name)}
              variant="danger"
            >
              <Trash2 size={15} />
            </Button>
          )}
          compact
        />
      </section>

      <section className="node-section">
        <h3>{t("Namespace isolation")}</h3>
        <form
          className="form-grid isolation-form"
          onSubmit={(event) => onSaveIsolation(namespace, event)}
        >
          <label className="check-row">
            <Checkbox
              name="enabled"
              type="checkbox"
              defaultChecked={Boolean(isolation.enabled)}
            />{" "}
            {t("Enable default deny isolation")}
          </label>
          <label className="check-row">
            <Checkbox
              name="allow_same_namespace"
              type="checkbox"
              defaultChecked={Boolean(isolation.allow_same_namespace)}
            />{" "}
            {t("Allow same namespace traffic")}
          </label>
          <label className="check-row">
            <Checkbox
              name="allow_dns"
              type="checkbox"
              defaultChecked={Boolean(isolation.allow_dns)}
            />{" "}
            {t("Allow DNS egress")}
          </label>
          <Button variant="primary">{t("Save isolation")}</Button>
        </form>
      </section>
    </div>
  );
}

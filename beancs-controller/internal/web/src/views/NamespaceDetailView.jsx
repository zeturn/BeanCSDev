import React, {useEffect, useMemo, useRef, useState} from "react";
import * as LucideIcons from "lucide-react";
import { formatTime, formatKeyValues } from "../utils/index";
import { MetricCard, SimpleTable } from "../components/index";
import {
  Activity, AlertTriangle, Bell, Boxes, Box, CheckCircle2, ChevronDown, ChevronRight, Cloud, Coffee, Code2, Container, Cpu, Database, Edit3, FileText, GitBranch, Github, Globe2, HardDrive, Image as ImageIcon, KeyRound, Layers3, LayoutDashboard, LineChart, ListRestart, LoaderCircle, Lock, Menu, MemoryStick, MoreHorizontal, Network, Package, Play, Plus, RefreshCw, RotateCcw, Rocket, ScrollText, Search, Server, Settings, Shield, ShieldCheck, Trash2, Upload, X
} from "lucide-react";
export default function NamespaceDetailView({detail, onPatchNamespace, onSaveResourceQuota, onDeleteResourceQuota, onSaveLimitRange, onDeleteLimitRange, onSavePermission, onDeletePermission, onSaveIsolation}) {
  const row = detail.row || {};
  const summary = row.summary || row;
  const namespace = summary.name || row.name;
  const stats = row.stats || {};
  const isolation = row.isolation || {};
  return (
    <div className="namespace-detail">
      {detail.loading && <p className="muted">Loading namespace detail...</p>}
      {detail.error && <p className="error-inline">{detail.error}</p>}
      <div className="dashboard-kpis">
        <MetricCard icon={Boxes} label="Pods" value={stats.pods || 0} detail={`${stats.running_pods || 0} running · ${stats.abnormal_pods || 0} abnormal`} />
        <MetricCard icon={Database} label="Services" value={stats.services || 0} detail={`${stats.deployments || 0} deployments`} />
        <MetricCard icon={Network} label="Ingresses" value={stats.ingresses || 0} detail={`${stats.network_policies || 0} policies`} />
        <MetricCard icon={KeyRound} label="Secrets" value={stats.secrets || 0} detail={`${stats.config_maps || 0} configmaps`} />
        <MetricCard icon={Shield} label="Isolation" value={isolation.enabled ? "On" : "Off"} detail={isolation.policy_name || "No default isolation"} tone={isolation.enabled ? "good" : "warning"} />
        <MetricCard icon={ListRestart} label="Checked" value={formatTime(row.checked_at)} detail={summary.status || "-"} />
      </div>

      <section className="node-section">
        <h3>Namespace labels</h3>
        <form className="form-grid node-edit-form" onSubmit={(event) => { event.preventDefault(); onPatchNamespace(namespace, event.currentTarget.labels.value); }}>
          <textarea name="labels" defaultValue={formatKeyValues(summary.labels)} />
          <button className="primary">Save labels</button>
        </form>
      </section>

      <section className="node-section">
        <h3>ResourceQuota</h3>
        <form className="form-grid quota-form" onSubmit={(event) => onSaveResourceQuota(namespace, event)}>
          <input name="name" placeholder="quota name" defaultValue="default-quota" required />
          <input name="hard" placeholder="requests.cpu=4,requests.memory=8Gi,limits.cpu=8,pods=40" required />
          <button className="primary">Save quota</button>
        </form>
        <SimpleTable rows={row.resource_quotas || []} columns={["name", "hard", "used"]} actions={(quota) => <button className="danger-button" onClick={() => onDeleteResourceQuota(namespace, quota.name)}><Trash2 size={15} /></button>} compact />
      </section>

      <section className="node-section">
        <h3>LimitRange</h3>
        <form className="form-grid limit-form" onSubmit={(event) => onSaveLimitRange(namespace, event)}>
          <input name="name" placeholder="limit range name" defaultValue="default-limits" required />
          <select name="type" defaultValue="Container"><option>Container</option><option>Pod</option><option>PersistentVolumeClaim</option></select>
          <input name="default_request" placeholder="default request: cpu=100m,memory=128Mi" />
          <input name="default" placeholder="default limit: cpu=500m,memory=512Mi" />
          <input name="min" placeholder="min: cpu=50m,memory=64Mi" />
          <input name="max" placeholder="max: cpu=2,memory=2Gi" />
          <button className="primary">Save limits</button>
        </form>
        <SimpleTable rows={row.limit_ranges || []} columns={["name", "types"]} actions={(limit) => <button className="danger-button" onClick={() => onDeleteLimitRange(namespace, limit.name)}><Trash2 size={15} /></button>} compact />
      </section>

      <section className="node-section">
        <h3>Namespace permissions</h3>
        <form className="form-grid permission-form" onSubmit={(event) => onSavePermission(namespace, event)}>
          <input name="name" placeholder="permission name" defaultValue="namespace-admin" required />
          <input name="subjects" placeholder="subjects: User:alice,Group:platform,ServiceAccount:builder" required />
          <input name="verbs" placeholder="verbs: get,list,watch,create,update,delete" defaultValue="get,list,watch" required />
          <input name="resources" placeholder="resources: pods,services,deployments" defaultValue="pods,services" required />
          <input name="api_groups" placeholder="api groups: ,apps,networking.k8s.io" />
          <button className="primary">Save permission</button>
        </form>
        <SimpleTable rows={row.role_bindings || []} columns={["name", "role_ref", "subjects"]} actions={(binding) => <button className="danger-button" onClick={() => onDeletePermission(namespace, binding.name)}><Trash2 size={15} /></button>} compact />
      </section>

      <section className="node-section">
        <h3>Namespace isolation</h3>
        <form className="form-grid isolation-form" onSubmit={(event) => onSaveIsolation(namespace, event)}>
          <label className="check-row"><input name="enabled" type="checkbox" defaultChecked={Boolean(isolation.enabled)} /> Enable default deny isolation</label>
          <label className="check-row"><input name="allow_same_namespace" type="checkbox" defaultChecked={Boolean(isolation.allow_same_namespace)} /> Allow same namespace traffic</label>
          <label className="check-row"><input name="allow_dns" type="checkbox" defaultChecked={Boolean(isolation.allow_dns)} /> Allow DNS egress</label>
          <button className="primary">Save isolation</button>
        </form>
      </section>
    </div>
  );
}

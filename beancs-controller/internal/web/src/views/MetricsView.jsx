import React, {useEffect, useMemo, useRef, useState} from "react";
import * as LucideIcons from "lucide-react";
import { formatTime, formatBytes, formatPercent } from "../utils/index";
import { MetricCard, IndustrialMeter } from "../components/index";
import {
  Activity, AlertTriangle, Bell, Boxes, Box, CheckCircle2, ChevronDown, ChevronRight, Cloud, Coffee, Code2, Container, Cpu, Database, Edit3, FileText, GitBranch, Github, Globe2, HardDrive, Image as ImageIcon, KeyRound, Layers3, LayoutDashboard, LineChart, ListRestart, LoaderCircle, Lock, Menu, MemoryStick, MoreHorizontal, Network, Package, Play, Plus, RefreshCw, RotateCcw, Rocket, ScrollText, Search, Server, Settings, Shield, ShieldCheck, Trash2, Upload, X
} from "lucide-react";
export default function MetricsView({dashboard, runtime, refresh}) {
  if (!dashboard) {
    return <section className="panel"><div className="empty">Loading metrics...</div></section>;
  }
  const resources = dashboard.resources || {};
  const nodes = runtime.nodes || [];
  return (
    <div className="stack observability-page">
      <section className="panel action-panel">
        <div>
          <h2><LineChart size={18} /> Metrics</h2>
          <p>Cluster capacity, utilization, and node-level resource readings from metrics-server and node stats.</p>
        </div>
        <button onClick={refresh}><RefreshCw size={15} /> Refresh</button>
      </section>
      <section className="dashboard-kpis">
        <MetricCard icon={Cpu} label="CPU" value={`${formatPercent(resources.cpu_percent)}%`} detail={`${resources.cpu_used_millis || 0}m / ${resources.cpu_total_millis || 0}m`} />
        <MetricCard icon={MemoryStick} label="Memory" value={`${formatPercent(resources.memory_percent)}%`} detail={`${formatBytes(resources.memory_used_bytes)} / ${formatBytes(resources.memory_total_bytes)}`} />
        <MetricCard icon={HardDrive} label="Disk" value={`${formatPercent(resources.disk_percent)}%`} detail={`${formatBytes(resources.disk_used_bytes)} / ${formatBytes(resources.disk_total_bytes)}`} />
        <MetricCard icon={Activity} label="Metrics source" value={dashboard.metrics_available ? "Live" : "Partial"} detail={dashboard.metrics_error || `Checked ${formatTime(dashboard.checked_at)}`} tone={dashboard.metrics_available ? "good" : "warning"} />
      </section>
      <section className="dashboard-grid">
        <div className="panel dashboard-panel">
          <h2><Activity size={18} /> Utilization</h2>
          <div className="industrial-meters">
            <IndustrialMeter label="CPU" value={resources.cpu_percent} detail={`${resources.cpu_used_millis || 0}m / ${resources.cpu_total_millis || 0}m`} />
            <IndustrialMeter label="Memory" value={resources.memory_percent} detail={`${formatBytes(resources.memory_used_bytes)} / ${formatBytes(resources.memory_total_bytes)}`} />
            <IndustrialMeter label="Disk" value={resources.disk_percent} detail={`${formatBytes(resources.disk_used_bytes)} / ${formatBytes(resources.disk_total_bytes)}`} />
          </div>
        </div>
        <div className="panel">
          <h2><Server size={18} /> Node readings</h2>
          <div className="mini-table">
            {nodes.map((node) => (
              <div key={node.name}>
                <span>{node.name}<small>{node.status || "-"} · {node.version || "-"}</small></span>
                <b>{node.cpu || node.cpu_percent || "-"} / {node.memory || node.memory_percent || "-"}</b>
              </div>
            ))}
            {nodes.length === 0 && <div className="empty">Node runtime data is not loaded yet.</div>}
          </div>
        </div>
      </section>
    </div>
  );
}

import React, {useEffect, useMemo, useRef, useState} from "react";
import * as LucideIcons from "lucide-react";
import { formatTime } from "../utils/index";
import { MetricCard, AlertList } from "../components/index";
import {
  Activity, AlertTriangle, Bell, Boxes, Box, CheckCircle2, ChevronDown, ChevronRight, Cloud, Coffee, Code2, Container, Cpu, Database, Edit3, FileText, GitBranch, Github, Globe2, HardDrive, Image as ImageIcon, KeyRound, Layers3, LayoutDashboard, LineChart, ListRestart, LoaderCircle, Lock, Menu, MemoryStick, MoreHorizontal, Network, Package, Play, Plus, RefreshCw, RotateCcw, Rocket, ScrollText, Search, Server, Settings, Shield, ShieldCheck, Trash2, Upload, X
} from "lucide-react";
export default function AlertsView({dashboard, refresh}) {
  if (!dashboard) {
    return <section className="panel"><div className="empty">Loading alerts...</div></section>;
  }
  const alerts = dashboard.alerts || [];
  const critical = alerts.filter((row) => ["critical", "error", "failed"].includes(String(row.severity || "").toLowerCase())).length;
  const warnings = alerts.length - critical;
  return (
    <div className="stack observability-page">
      <section className="panel action-panel">
        <div>
          <h2><AlertTriangle size={18} /> Alerts</h2>
          <p>Active cluster health signals generated from abnormal pods, warning events, and node readiness.</p>
        </div>
        <button onClick={refresh}><RefreshCw size={15} /> Refresh</button>
      </section>
      <section className="dashboard-kpis">
        <MetricCard icon={AlertTriangle} label="Active" value={alerts.length} detail={`${critical} critical · ${warnings} warning`} tone={alerts.length > 0 ? "warning" : "good"} />
        <MetricCard icon={Server} label="Nodes" value={`${dashboard.nodes?.ready || 0}/${dashboard.nodes?.total || 0}`} detail={`${dashboard.nodes?.not_ready || 0} not ready`} tone={dashboard.nodes?.not_ready ? "warning" : "good"} />
        <MetricCard icon={Boxes} label="Pods" value={dashboard.pods?.abnormal || 0} detail={`${dashboard.pods?.pending || 0} pending · ${dashboard.pods?.failed || 0} failed`} tone={dashboard.pods?.abnormal ? "warning" : "good"} />
        <MetricCard icon={Activity} label="Status" value={dashboard.status || "-"} detail={`Last check ${formatTime(dashboard.checked_at)}`} tone={dashboard.healthy ? "good" : "warning"} />
      </section>
      <section className="panel">
        <h2><Shield size={18} /> Alert feed</h2>
        <AlertList rows={alerts} empty="No active alerts reported." />
      </section>
    </div>
  );
}

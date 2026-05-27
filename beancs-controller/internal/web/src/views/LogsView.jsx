import React, {useEffect, useMemo, useRef, useState} from "react";
import * as LucideIcons from "lucide-react";
import { MetricCard } from "../components/index";
import {
  Activity, AlertTriangle, Bell, Boxes, Box, CheckCircle2, ChevronDown, ChevronRight, Cloud, Coffee, Code2, Container, Cpu, Database, Edit3, FileText, GitBranch, Github, Globe2, HardDrive, Image as ImageIcon, KeyRound, Layers3, LayoutDashboard, LineChart, ListRestart, LoaderCircle, Lock, Menu, MemoryStick, MoreHorizontal, Network, Package, Play, Plus, RefreshCw, RotateCcw, Rocket, ScrollText, Search, Server, Settings, Shield, ShieldCheck, Trash2, Upload, X
} from "lucide-react";
export default function LogsView({projects, activeProjectID, setActiveProjectID, progress, refresh, logFollow, liveLogs, logStatus, onStartLogFollow, onStopLogFollow, onOpenPods}) {
  const logs = logFollow ? liveLogs : progress?.logs;
  return (
    <div className="stack observability-page">
      <section className="panel action-panel">
        <div>
          <h2><ScrollText size={18} /> Logs</h2>
          <p>Project container log snapshots and live follow without leaving the observability section.</p>
        </div>
        <div className="progress-controls">
          <select value={activeProjectID} onChange={(event) => setActiveProjectID(event.target.value)}>
            <option value="">Choose project</option>
            {projects.map((project) => <option key={project.id} value={project.id}>{project.display_name || project.name}</option>)}
          </select>
          <button onClick={() => refresh()} disabled={logFollow}><RefreshCw size={15} /> Snapshot</button>
          {logFollow ? <button onClick={onStopLogFollow}>Stop follow</button> : <button className="primary" onClick={() => onStartLogFollow(activeProjectID)} disabled={!activeProjectID}>Follow live</button>}
        </div>
      </section>
      <section className="dashboard-kpis">
        <MetricCard icon={Boxes} label="Project" value={progress?.project?.display_name || progress?.project?.name || "-"} detail={progress?.project?.namespace || "No project selected"} />
        <MetricCard icon={Layers3} label="Pods" value={(progress?.pods || []).length} detail={`${(progress?.pods || []).filter((pod) => pod.status === "Running").length} running`} />
        <MetricCard icon={GitBranch} label="Deployments" value={(progress?.deployments || []).length} detail={(progress?.deployments || [])[0]?.status || "No deployment events"} />
      </section>
      <section className="panel log-panel observability-log-panel">
        <div className="log-header">
          <h2><Code2 size={18} /> Container logs</h2>
          <div className="row-actions">
            <button type="button" onClick={onOpenPods}><Layers3 size={15} /> Pod detail</button>
          </div>
        </div>
        {logStatus && <p className="log-status">{logStatus}</p>}
        <pre>{logs || "Choose a project to load recent logs."}</pre>
      </section>
    </div>
  );
}

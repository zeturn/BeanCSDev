import React, {useEffect, useMemo, useRef, useState} from "react";
import * as LucideIcons from "lucide-react";
import { formatDeploymentDate, shortRelativeDuration, normalizeDeploymentStatus, imageRepoName, deploymentShortID, truncateMiddle } from "../utils/index";
import {
  Activity, AlertTriangle, Bell, Boxes, Box, CheckCircle2, ChevronDown, ChevronRight, Cloud, Coffee, Code2, Container, Cpu, Database, Edit3, FileText, GitBranch, Github, Globe2, HardDrive, Image as ImageIcon, KeyRound, Layers3, LayoutDashboard, LineChart, ListRestart, LoaderCircle, Lock, Menu, MemoryStick, MoreHorizontal, Network, Package, Play, Plus, RefreshCw, RotateCcw, Rocket, ScrollText, Search, Server, Settings, Shield, ShieldCheck, Trash2, Upload, X
} from "lucide-react";
export default function DeploymentsView({projects, processes, runtimeDeployments, refresh, onOpenProcess}) {
  const processRows = (processes || []).filter((process) => process.type === "deployment").map((process) => {
    const project = process.project || (projects || []).find((row) => row.id === process.project_id) || {};
    const deployment = process.deployment || {};
    const status = normalizeDeploymentStatus(process.status);
    return {
      id: process.id,
      process,
      name: project.display_name || project.name || process.title,
      environment: project.namespace || "default",
      current: process.status === "succeeded",
      status,
      duration: process.updated_at ? shortRelativeDuration(process.updated_at) : "-",
      repo: project.github_repo || imageRepoName(deployment.image_ref || project.image_reference) || project.build_source || "registry",
      branch: project.github_branch || "main",
      commit: deployment.image_ref || deployment.commit_sha || deployment.tag || process.failure_reason || "-",
      created: process.created_at,
      author: process.triggered_by || "-",
    };
  });
  const fallbackRows = processRows.length ? [] : (runtimeDeployments || []).map((deployment, index) => ({
    id: deployment.uid || deployment.name || index,
    name: deployment.name || `deployment-${index + 1}`,
    environment: deployment.namespace || "default",
    current: false,
    status: normalizeDeploymentStatus(deployment.ready_replicas === deployment.replicas ? "ready" : deployment.status),
    duration: deployment.age || "-",
    repo: deployment.image || deployment.name || "-",
    branch: deployment.namespace || "default",
    commit: deployment.strategy || deployment.status || "-",
    created: deployment.created_at || deployment.updated_at,
    author: "cluster",
  }));
  const rows = processRows.length ? processRows : fallbackRows;
  return (
    <section className="deployments-page">
      <div className="deployment-list">
        {rows.map((row) => (
          <button type="button" className="deployment-row" key={row.id} onClick={() => row.process && onOpenProcess?.(row.process)}>
            <span className="deploy-id">
              <b>{deploymentShortID(row.name, row.id)}</b>
              <small>{row.environment}{row.current && <em>Current</em>}</small>
            </span>
            <span className={`deploy-state ${row.status}`}>
              <i />
              <b>{row.status === "error" ? "Error" : row.status === "building" ? "Building" : "Ready"}</b>
              <small>{row.duration}</small>
            </span>
            <span className="deploy-repo">
              <span className="repo-mark">B</span>
              <b>{row.repo}</b>
            </span>
            <span className="deploy-branch">
              <span><GitBranch size={18} /> <b>{row.branch}</b></span>
              <small>{truncateMiddle(row.commit, 34)}</small>
            </span>
            <span className="deploy-meta">{formatDeploymentDate(row.created)} by {row.author}<span className="mini-avatar" /></span>
          </button>
        ))}
        {rows.length === 0 && <div className="empty">No deployments found.</div>}
      </div>
    </section>
  );
}

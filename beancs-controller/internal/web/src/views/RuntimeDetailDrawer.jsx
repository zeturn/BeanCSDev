import React, {useEffect, useMemo, useRef, useState} from "react";
import * as LucideIcons from "lucide-react";
import { detailTitle, formatCell, formatKeyValues } from "../utils/index";
import { ContainerLogViewer, ServiceForm } from "../components/index";
import {
  Activity, AlertTriangle, Bell, Boxes, Box, CheckCircle2, ChevronDown, ChevronRight, Cloud, Coffee, Code2, Container, Cpu, Database, Edit3, FileText, GitBranch, Github, Globe2, HardDrive, Image as ImageIcon, KeyRound, Layers3, LayoutDashboard, LineChart, ListRestart, LoaderCircle, Lock, Menu, MemoryStick, MoreHorizontal, Network, Package, Play, Plus, RefreshCw, RotateCcw, Rocket, ScrollText, Search, Server, Settings, Shield, ShieldCheck, Trash2, Upload, X
} from "lucide-react";
export default function RuntimeDetailDrawer({detail, logs, logFollow, logStatus, selectedLogContainer, logTail, logLoaded, nodeHealth, onLoadNodeHealth, onSaveNodeLabels, onSaveNodeTaints, onCordonNode, onDrainNode, onDeleteNode, onSaveResourceQuota, onDeleteResourceQuota, onSaveLimitRange, onDeleteLimitRange, onSaveNamespacePermission, onDeleteNamespacePermission, onSaveNamespaceIsolation, onSelectLogContainer, onSetLogTail, onLoadContainerLogs, onFollowPodLogs, onStopPodLogs, onClose, onSaveService, onPatchNamespace}) {
  const row = detail.row || {};
  const title = `${detailTitle(detail.kind)} · ${row.namespace ? `${row.namespace}/` : ""}${row.name || row.summary?.name || ""}`;
  return (
    <div className="side-drawer-backdrop" onClick={onClose}>
      <aside className="side-drawer runtime-detail-drawer" onClick={(event) => event.stopPropagation()}>
        <div className="side-drawer-head">
          <div>
            <h2>{title}</h2>
            <p>{detail.loading ? "Loading live details..." : detail.error || "Live Kubernetes resource detail"}</p>
          </div>
          <button type="button" className="icon-button" aria-label="Close" onClick={onClose}><X size={16} /></button>
        </div>
        {detail.kind === "service-edit" ? (
          <ServiceForm existing={row} onSubmit={(event) => onSaveService(event, row)} />
        ) : detail.kind === "namespaces" ? (
          <form className="form-grid" onSubmit={(event) => { event.preventDefault(); onPatchNamespace(row.name, event.currentTarget.labels.value); onClose(); }}>
            <label>Labels</label>
            <textarea name="labels" defaultValue={formatKeyValues(row.labels)} />
            <button className="primary">Save labels</button>
          </form>
        ) : detail.kind === "pod" ? (
          <>
            <ContainerLogViewer
              pod={row}
              logs={logs}
              logFollow={logFollow}
              logStatus={logStatus}
              selectedContainer={selectedLogContainer}
              tail={logTail}
              loaded={logLoaded}
              onSelectContainer={onSelectLogContainer}
              onSetTail={onSetLogTail}
              onLoad={() => onLoadContainerLogs(row, selectedLogContainer, logTail)}
              onFollow={() => onFollowPodLogs(row, selectedLogContainer, logTail)}
              onStop={onStopPodLogs}
            />
          </>
        ) : detail.kind === "node" ? (
          <NodeDetailView detail={detail} health={nodeHealth} onLoadHealth={onLoadNodeHealth} onSaveLabels={onSaveNodeLabels} onSaveTaints={onSaveNodeTaints} onCordon={onCordonNode} onDrain={onDrainNode} onDelete={onDeleteNode} />
        ) : detail.kind === "namespace-detail" ? (
          <NamespaceDetailView detail={detail} onPatchNamespace={onPatchNamespace} onSaveResourceQuota={onSaveResourceQuota} onDeleteResourceQuota={onDeleteResourceQuota} onSaveLimitRange={onSaveLimitRange} onDeleteLimitRange={onDeleteLimitRange} onSavePermission={onSaveNamespacePermission} onDeletePermission={onDeleteNamespacePermission} onSaveIsolation={onSaveNamespaceIsolation} />
        ) : (
          <div className="detail-list">{Object.entries(row).map(([key, value]) => <span key={key}>{key.replaceAll("_", " ")} <b>{formatCell(value)}</b></span>)}</div>
        )}
      </aside>
    </div>
  );
}

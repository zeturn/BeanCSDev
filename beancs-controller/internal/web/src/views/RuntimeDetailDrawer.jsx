import React, { useEffect, useMemo, useRef, useState } from "react";
import * as LucideIcons from "lucide-react";
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
import * as Utils from "../utils";
import * as API from "../api";
import * as Components from "../components";
import { detailTitle, formatCell, formatKeyValues } from "../utils/index";
import { ContainerLogViewer, ServiceForm } from "../components/index";
import NodeDetailView from "./NodeDetailView";
import NamespaceDetailView from "./NamespaceDetailView";
import { Drawer, Button, Textarea } from "../components/index";
import { t } from "../i18n/index";
export default function RuntimeDetailDrawer({
  detail,
  logs,
  logFollow,
  logStatus,
  selectedLogContainer,
  logTail,
  logLoaded,
  nodeHealth,
  onLoadNodeHealth,
  onSaveNodeLabels,
  onSaveNodeTaints,
  onCordonNode,
  onDrainNode,
  onDeleteNode,
  onSaveResourceQuota,
  onDeleteResourceQuota,
  onSaveLimitRange,
  onDeleteLimitRange,
  onSaveNamespacePermission,
  onDeleteNamespacePermission,
  onSaveNamespaceIsolation,
  onSelectLogContainer,
  onSetLogTail,
  onLoadContainerLogs,
  onFollowPodLogs,
  onStopPodLogs,
  onClose,
  onSaveService,
  onPatchNamespace,
}) {
  const row = detail.row || {};
  const title = `${detailTitle(detail.kind)} · ${row.namespace ? `${row.namespace}/` : ""}${row.name || row.summary?.name || ""}`;
  return (
    <Drawer
      className="runtime-detail-drawer"
      title={title}
      subtitle={
        detail.loading
          ? t("Loading live details...")
          : detail.error || t("Live Kubernetes resource detail")
      }
      onClose={onClose}
    >
      {detail.kind === "service-edit" ? (
        <ServiceForm
          existing={row}
          onSubmit={(event) => onSaveService(event, row)}
        />
      ) : detail.kind === "namespaces" ? (
        <form
          className="form-grid"
          onSubmit={(event) => {
            event.preventDefault();
            onPatchNamespace(row.name, event.currentTarget.labels.value);
            onClose();
          }}
        >
          <label>{t("Labels")}</label>
          <Textarea name="labels" defaultValue={formatKeyValues(row.labels)} />
          <Button variant="primary">{t("Save labels")}</Button>
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
            onLoad={() =>
              onLoadContainerLogs(row, selectedLogContainer, logTail)
            }
            onFollow={() => onFollowPodLogs(row, selectedLogContainer, logTail)}
            onStop={onStopPodLogs}
          />
        </>
      ) : detail.kind === "node" ? (
        <NodeDetailView
          detail={detail}
          health={nodeHealth}
          onLoadHealth={onLoadNodeHealth}
          onSaveLabels={onSaveNodeLabels}
          onSaveTaints={onSaveNodeTaints}
          onCordon={onCordonNode}
          onDrain={onDrainNode}
          onDelete={onDeleteNode}
        />
      ) : detail.kind === "namespace-detail" ? (
        <NamespaceDetailView
          detail={detail}
          onPatchNamespace={onPatchNamespace}
          onSaveResourceQuota={onSaveResourceQuota}
          onDeleteResourceQuota={onDeleteResourceQuota}
          onSaveLimitRange={onSaveLimitRange}
          onDeleteLimitRange={onDeleteLimitRange}
          onSavePermission={onSaveNamespacePermission}
          onDeletePermission={onDeleteNamespacePermission}
          onSaveIsolation={onSaveNamespaceIsolation}
        />
      ) : (
        <div className="table network-table runtime-detail-table">
          <div className="tr head">
            <span>{t("Key")}</span>
            <span>{t("Value")}</span>
          </div>
          {Object.entries(row).map(([k, v]) => (
            <div className="tr" key={k}>
              <span>
                <b>{k.replaceAll("_", " ")}</b>
              </span>
              <span>{formatCell(v)}</span>
            </div>
          ))}
        </div>
      )}
    </Drawer>
  );
}

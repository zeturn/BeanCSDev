import React, { useEffect, useMemo, useRef, useState } from "react";
import * as LucideIcons from "lucide-react";
import { t } from "../i18n/index";
import { formatTime } from "../utils/index";
import { EventTimeline, MetricCard, Button } from "../components/index";
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
export default function EventsView({ dashboard, refresh }) {
  if (!dashboard) {
    return (
      <section className="panel">
        <div className="empty">{t("Loading events...")}</div>
      </section>
    );
  }
  const events = dashboard.events || [];
  const byReason = events.reduce((acc, event) => {
    const key = event.reason || event.type || "Unknown";
    acc[key] = (acc[key] || 0) + Number(event.count || 1);
    return acc;
  }, {});
  return (
    <div className="stack observability-page">
      <section className="panel action-panel">
        <div>
          <h2>
            <ListRestart size={18} /> {t("Events")}
          </h2>
          <p>
            {t(
              "Recent warning events from the Kubernetes event stream, grouped by object, reason, and last seen time.",
            )}
          </p>
        </div>
        <Button onClick={refresh}>
          <RefreshCw size={15} /> {t("Refresh")}
        </Button>
      </section>
      <section className="dashboard-kpis">
        <MetricCard
          icon={ListRestart}
          label={t("Warning events")}
          value={events.length}
          detail={t("{count} reasons", {
            count: Object.keys(byReason).length,
          })}
          tone={events.length > 0 ? "warning" : "good"}
        />
        <MetricCard
          icon={AlertTriangle}
          label={t("Event count")}
          value={events.reduce(
            (sum, event) => sum + Number(event.count || 1),
            0,
          )}
          detail={t("Summed Kubernetes count values")}
        />
        <MetricCard
          icon={Activity}
          label={t("Cluster")}
          value={dashboard.status || "-"}
          detail={t("Checked {time}", {
            time: formatTime(dashboard.checked_at),
          })}
          tone={dashboard.healthy ? "good" : "warning"}
        />
      </section>
      <section className="dashboard-grid">
        <div className="panel">
          <h2>
            <Database size={18} /> {t("Reasons")}
          </h2>
          <div className="mini-table">
            {Object.entries(byReason).map(([reason, count]) => (
              <div key={reason}>
                <span>{reason}</span>
                <b>{count}</b>
              </div>
            ))}
            {Object.keys(byReason).length === 0 && (
              <div className="empty">
                {t("No warning reasons in the latest feed.")}
              </div>
            )}
          </div>
        </div>
        <div className="panel">
          <h2>
            <ScrollText size={18} /> {t("Event stream")}
          </h2>
          <EventTimeline events={events} />
        </div>
      </section>
    </div>
  );
}

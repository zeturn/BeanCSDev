import React, { useEffect, useMemo, useRef, useState } from "react";
import * as LucideIcons from "lucide-react";
import { formatTime } from "../utils/index";
import { ExpandableCell, Button } from "../components/index";
import { t } from "../i18n/index";
import APIKeyDrawer from "./APIKeyDrawer";
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
export default function APIKeysView({
  keys,
  scopeCatalog,
  createdKey,
  onDismissCreated,
  onCreate,
  onRevoke,
  onRefresh,
}) {
  const presets = scopeCatalog?.presets || [];
  const scopes = scopeCatalog?.scopes || [];
  const [drawerOpen, setDrawerOpen] = useState(false);
  return (
    <div className="stack">
      <section className="panel action-panel">
        <div>
          <h2>
            <KeyRound size={18} /> {t("API keys")}
          </h2>
          <p>
            {t(
              "Create keys for beanctl, scripts, and external systems that need to manage BeanCS through the API.",
            )}
          </p>
        </div>
        <div className="row-actions">
          <Button onClick={onRefresh}>
            <RefreshCw size={15} /> {t("Refresh")}
          </Button>
          <Button
            type="button"
            onClick={() => setDrawerOpen(true)}
            variant="primary"
          >
            <Plus size={15} /> {t("Create key")}
          </Button>
        </div>
      </section>
      {createdKey && (
        <section className="panel api-key-created">
          <h2>
            <Shield size={18} /> {t("Save this API key now")}
          </h2>
          <p className="muted">
            {t("BeanCS stores only a hash. This full key will not be shown again.")}
          </p>
          <pre>{createdKey.key}</pre>
          <div className="modal-actions">
            <Button onClick={onDismissCreated}>{t("I saved it")}</Button>
          </div>
        </section>
      )}
      <section className="panel">
          <h2>
            <KeyRound size={18} /> {t("Issued keys")}
          </h2>
          <div className="table api-key-table">
            <div className="tr head">
              <span>{t("Name")}</span>
              <span>{t("Prefix")}</span>
              <span>{t("Scopes")}</span>
              <span>{t("Last used")}</span>
              <span>{t("Expires")}</span>
              <span>{t("Actions")}</span>
            </div>
          {keys.map((key) => (
            <div className="tr" key={key.id}>
              <ExpandableCell className="strong" value={key.name} max={32} />
              <ExpandableCell value={key.prefix} max={24} />
              <ExpandableCell
                value={(key.scopes || []).join(", ") || "-"}
                max={38}
              />
              <ExpandableCell value={formatTime(key.last_used_at)} max={28} />
              <ExpandableCell
                value={
                  key.revoked_at
                    ? `Revoked ${formatTime(key.revoked_at)}`
                    : formatTime(key.expires_at)
                }
                max={32}
              />
              <span className="row-actions">
                <Button
                  disabled={Boolean(key.revoked_at)}
                  onClick={() => onRevoke(key)}
                  variant="danger"
                >
                  <Trash2 size={15} /> {t("Revoke")}
                </Button>
              </span>
            </div>
          ))}
          {keys.length === 0 && (
            <div className="empty">{t("No API keys issued yet.")}</div>
          )}
        </div>
      </section>
      {drawerOpen && (
        <APIKeyDrawer
          presets={presets}
          scopes={scopes}
          onClose={() => setDrawerOpen(false)}
          onCreate={async (event) => {
            const ok = await onCreate(event);
            if (ok) setDrawerOpen(false);
          }}
        />
      )}
    </div>
  );
}

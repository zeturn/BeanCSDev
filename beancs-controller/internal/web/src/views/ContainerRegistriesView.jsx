import React, { useEffect, useMemo, useRef, useState } from "react";
import * as LucideIcons from "lucide-react";
import { formatTime } from "../utils/index";
import {
  ExpandableCell,
  Button,
  Modal,
  PaginationBar,
  Select,
  Input,
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
export default function ContainerRegistriesView({
  presets,
  registries,
  images,
  onAddRegistry,
  onDeleteRegistry,
  onAddImage,
  onRefreshImage,
  onDeleteImage,
  onSyncAll,
  onRefresh,
}) {
  const presetByKind = useMemo(
    () => Object.fromEntries((presets || []).map((p) => [p.kind, p])),
    [presets],
  );
  const [previewKind, setPreviewKind] = useState("ghcr");
  const [registryCreateOpen, setRegistryCreateOpen] = useState(false);
  const [imageCreateOpen, setImageCreateOpen] = useState(false);
  const [registryPage, setRegistryPage] = useState(1);
  const [imagePage, setImagePage] = useState(1);
  const registrySize = 10;
  const imageSize = 8;
  const pagedRegistries = (registries || []).slice(
    (registryPage - 1) * registrySize,
    registryPage * registrySize,
  );
  const pagedImages = (images || []).slice(
    (imagePage - 1) * imageSize,
    imagePage * imageSize,
  );
  useEffect(() => setRegistryPage(1), [(registries || []).length]);
  useEffect(() => setImagePage(1), [(images || []).length]);
  return (
    <div className="stack registry-page">
      <section className="panel action-panel">
        <div>
          <h2>
            <Package size={18} /> {t("Image registries")}
          </h2>
          <p>
            {t(
              "Lists tags via the Docker Registry HTTP API V2; Docker Hub uses registry-1.docker.io. Provide credentials for private registries.",
            )}
          </p>
        </div>
        <Button type="button" onClick={onRefresh}>
          <RefreshCw size={15} /> {t("Refresh")}
        </Button>
      </section>

      <section className="panel action-panel">
        <div>
          <h2>
            <Database size={18} /> {t("Manage registries and image tracking")}
          </h2>
          <p className="muted">
            {t("Creation entry separated into dialogs; the list stays light.")}
          </p>
        </div>
        <div className="row-actions">
          <Button type="button" variant="primary" onClick={() => setRegistryCreateOpen(true)}>
            <Plus size={15} /> {t("Add registry")}
          </Button>
          <Button type="button" onClick={() => setImageCreateOpen(true)}>
            <Plus size={15} /> {t("Add image tracking")}
          </Button>
        </div>
      </section>

      <section className="panel">
        <h2>
          <Database size={18} /> {t("Saved registries")}
        </h2>
        <div className="table compact-table registry-table">
          <div className="tr head">
            <span>{t("Name")}</span>
            <span>{t("Type")}</span>
            <span>{t("API root")}</span>
            <span>{t("Auth")}</span>
            <span />
          </div>
          {pagedRegistries.map((r) => (
            <div className="tr" key={r.id}>
              <ExpandableCell className="strong" value={r.name} max={30} />
              <ExpandableCell value={r.kind} max={24} />
              <ExpandableCell className="mono" value={r.api_base} max={42} />
              <ExpandableCell
                value={r.has_auth ? t("Configured") : t("Anonymous")}
                max={24}
              />
              <span className="row-actions">
                <Button
                  type="button"
                  onClick={() => onDeleteRegistry(r)}
                  variant="danger"
                >
                  <Trash2 size={15} /> {t("Delete")}
                </Button>
              </span>
            </div>
          ))}
          {(registries || []).length === 0 && (
            <div className="empty">{t("No registries added yet.")}</div>
          )}
        </div>
        <PaginationBar
          page={registryPage}
          pageSize={registrySize}
          total={(registries || []).length}
          onPageChange={setRegistryPage}
          label="registries"
        />
      </section>

      <section className="panel">
        <div className="panel-heading-inline">
          <h2>
            <Boxes size={18} /> {t("Images and tags")}
          </h2>
          <Button
            type="button"
            onClick={onSyncAll}
            disabled={!(images || []).length}
            variant="ghost"
          >
            <RefreshCw size={15} /> {t("Sync all remote tags")}
          </Button>
        </div>
        <p className="muted">
          {t(
            "Repository paths must match the Registry API (e.g. Docker Hub official nginx:",
          )}{" "}
          <span className="mono">library/nginx</span>; GHCR:{" "}
          <span className="mono">owner/repo</span>
          {t(
            "). Tags are fetched immediately after saving; the local cache refreshes every 2 minutes.",
          )}
        </p>
        {pagedImages.map((im) => (
          <div className="registry-image-card" key={im.id}>
            <div className="registry-image-head">
              <div>
                <div className="mono strong">{im.repository}</div>
                <small className="muted">
                  {t("Source")}: {im.registry?.name || `registry #${im.registry_id}`} ·{" "}
                  {t("updated")} {formatTime(im.refreshed_at)}
                </small>
              </div>
              <div className="row-actions">
                <Button type="button" onClick={() => onRefreshImage(im.id)}>
                  <RefreshCw size={15} /> {t("Sync tags")}
                </Button>
                <Button
                  type="button"
                  onClick={() => onDeleteImage(im)}
                  variant="danger"
                >
                  <Trash2 size={15} /> {t("Remove")}
                </Button>
              </div>
            </div>
            <div className="tag-chip-grid">
              {(im.tags || []).slice(0, 200).map((tg) => (
                <span className="tag-chip" key={tg}>
                  {tg}
                </span>
              ))}
              {(im.tags || []).length > 200 && (
                <span className="muted">
                  … {t("{count} tags total, showing first 200", { count: (im.tags || []).length })}
                </span>
              )}
              {(im.tags || []).length === 0 && (
                <span className="muted">{t("No tags yet or sync failed.")}</span>
              )}
            </div>
          </div>
        ))}
        {(images || []).length === 0 && (
          <div className="empty">{t("No image registry tracking added yet.")}</div>
        )}
        <PaginationBar
          page={imagePage}
          pageSize={imageSize}
          total={(images || []).length}
          onPageChange={setImagePage}
          label="tracked images"
        />
      </section>
      {registryCreateOpen && (
        <Modal
          title={t("Add registry")}
          subtitle={t("Create form moved into a dialog to shorten the main page.")}
          onClose={() => setRegistryCreateOpen(false)}
        >
          <form
            className="form-grid registry-form"
            onSubmit={async (event) => {
              await onAddRegistry(event);
              setRegistryCreateOpen(false);
            }}
          >
            <label>
              {t("Type")}
              <Select
                name="kind"
                value={previewKind}
                onChange={(e) => setPreviewKind(e.target.value)}
              >
                {(presets || []).map((p) => (
                  <option key={p.kind} value={p.kind}>
                    {p.label}
                  </option>
                ))}
              </Select>
            </label>
            <label>
              {t("Display name (optional)")}
              <Input
                name="name"
                placeholder={`${t("e.g.")} ${presetByKind[previewKind]?.label || ""}`}
              />
            </label>
            <label className="span-2">
              {t("Registry address")}
              <Input
                name="host"
                required
                placeholder={
                  presetByKind[previewKind]?.example_host ||
                  "registry.example.com"
                }
              />
            </label>
            <label>
              {t("Username (optional)")}
              <Input
                name="username"
                autoComplete="off"
                placeholder={t("Private registry / PAT username")}
              />
            </label>
            <label>
              {t("Password or token (optional)")}
              <Input
                name="password"
                type="password"
                autoComplete="new-password"
                placeholder={t("Not stored in plaintext")}
              />
            </label>
            <label className="checkbox-row span-2">
              <Checkbox name="insecure_tls" type="checkbox" />
              {t("Skip TLS verification (trusted internal network only)")}
            </label>
            {presetByKind[previewKind]?.hint && (
              <p className="muted span-2">{presetByKind[previewKind].hint}</p>
            )}
            <div className="modal-actions span-2">
              <Button type="button" onClick={() => setRegistryCreateOpen(false)}>
                {t("Cancel")}
              </Button>
              <Button type="submit" variant="primary">
                <Plus size={15} /> {t("Save registry")}
              </Button>
            </div>
          </form>
        </Modal>
      )}
      {imageCreateOpen && (
        <Modal
          title={t("Add image tracking")}
          subtitle={t("Separate creation from viewing to reduce single-page density.")}
          onClose={() => setImageCreateOpen(false)}
        >
          <form
            className="form-grid registry-form"
            onSubmit={async (event) => {
              await onAddImage(event);
              setImageCreateOpen(false);
            }}
          >
            <label>
              {t("Registry")}
              <Select name="registry_id" required>
                <option value="">{t("Choose...")}</option>
                {(registries || []).map((r) => (
                  <option key={r.id} value={r.id}>
                    {r.name} ({r.kind})
                  </option>
                ))}
              </Select>
            </label>
            <label className="span-2">
              {t("Repository path")}
              <Input name="repository" required placeholder="namespace/name" />
            </label>
            <div className="modal-actions span-2">
              <Button type="button" onClick={() => setImageCreateOpen(false)}>
                {t("Cancel")}
              </Button>
              <Button type="submit" variant="primary">
                <Plus size={15} /> {t("Add and sync tags")}
              </Button>
            </div>
          </form>
        </Modal>
      )}
    </div>
  );
}

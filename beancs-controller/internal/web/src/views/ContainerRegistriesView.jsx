import React, { useEffect, useMemo, useRef, useState } from "react";
import * as LucideIcons from "lucide-react";
import { formatTime } from "../utils/index";
import {
  ExpandableCell,
  Button,
  Select,
  Input,
  Checkbox,
} from "../components/index";
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
  return (
    <div className="stack registry-page">
      <section className="panel action-panel">
        <div>
          <h2>
            <Package size={18} /> 镜像源
          </h2>
          <p>
            基于 Docker Registry HTTP API V2 列出标签；Docker Hub 会使用
            registry-1.docker.io；私有仓库请填写凭据。
          </p>
        </div>
        <Button type="button" onClick={onRefresh}>
          <RefreshCw size={15} /> 刷新
        </Button>
      </section>

      <section className="panel">
        <h2>
          <Plus size={18} /> 添加镜像源
        </h2>
        <form className="form-grid registry-form" onSubmit={onAddRegistry}>
          <label>
            类型
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
            显示名称（可选）
            <Input
              name="name"
              placeholder={`例如 ${presetByKind[previewKind]?.label || ""}`}
            />
          </label>
          <label className="span-2">
            镜像源地址
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
            用户名（可选）
            <Input
              name="username"
              autoComplete="off"
              placeholder="私有仓库 / PAT 用户名"
            />
          </label>
          <label>
            密码或 Token（可选）
            <Input
              name="password"
              type="password"
              autoComplete="new-password"
              placeholder="不会明文存储"
            />
          </label>
          <label className="checkbox-row span-2">
            <Checkbox name="insecure_tls" type="checkbox" />
            跳过 TLS 校验（仅可信内网）
          </label>
          {presetByKind[previewKind]?.hint && (
            <p className="muted span-2">{presetByKind[previewKind].hint}</p>
          )}
          <Button type="submit" variant="primary">
            <Plus size={15} /> 保存镜像源
          </Button>
        </form>
      </section>

      <section className="panel">
        <h2>
          <Database size={18} /> 已保存的镜像源
        </h2>
        <div className="table compact-table registry-table">
          <div className="tr head">
            <span>名称</span>
            <span>类型</span>
            <span>API 根</span>
            <span>鉴权</span>
            <span />
          </div>
          {(registries || []).map((r) => (
            <div className="tr" key={r.id}>
              <ExpandableCell className="strong" value={r.name} max={30} />
              <ExpandableCell value={r.kind} max={24} />
              <ExpandableCell className="mono" value={r.api_base} max={42} />
              <ExpandableCell value={r.has_auth ? "已配置" : "匿名"} max={24} />
              <span className="row-actions">
                <Button
                  type="button"
                  onClick={() => onDeleteRegistry(r)}
                  variant="danger"
                >
                  <Trash2 size={15} /> 删除
                </Button>
              </span>
            </div>
          ))}
          {(registries || []).length === 0 && (
            <div className="empty">尚未添加镜像源。</div>
          )}
        </div>
      </section>

      <section className="panel">
        <div className="panel-heading-inline">
          <h2>
            <Boxes size={18} /> 镜像与标签
          </h2>
          <Button
            type="button"
            onClick={onSyncAll}
            disabled={!(images || []).length}
            variant="ghost"
          >
            <RefreshCw size={15} /> 同步全部远程标签
          </Button>
        </div>
        <p className="muted">
          仓库路径需与 Registry API 一致（例如 Docker Hub 官方 nginx：
          <span className="mono">library/nginx</span>；GHCR：
          <span className="mono">owner/repo</span>
          ）。保存后会立即拉取标签；页面每 2 分钟刷新本地缓存列表。
        </p>
        <form className="form-grid registry-form" onSubmit={onAddImage}>
          <label>
            镜像源
            <Select name="registry_id" required>
              <option value="">选择...</option>
              {(registries || []).map((r) => (
                <option key={r.id} value={r.id}>
                  {r.name} ({r.kind})
                </option>
              ))}
            </Select>
          </label>
          <label className="span-2">
            仓库路径（repository）
            <Input name="repository" required placeholder="namespace/name" />
          </label>
          <Button type="submit" variant="primary">
            <Plus size={15} /> 添加并同步标签
          </Button>
        </form>

        {(images || []).map((im) => (
          <div className="registry-image-card" key={im.id}>
            <div className="registry-image-head">
              <div>
                <div className="mono strong">{im.repository}</div>
                <small className="muted">
                  来源：{im.registry?.name || `registry #${im.registry_id}`} ·
                  更新 {formatTime(im.refreshed_at)}
                </small>
              </div>
              <div className="row-actions">
                <Button type="button" onClick={() => onRefreshImage(im.id)}>
                  <RefreshCw size={15} /> 同步标签
                </Button>
                <Button
                  type="button"
                  onClick={() => onDeleteImage(im)}
                  variant="danger"
                >
                  <Trash2 size={15} /> 移除
                </Button>
              </div>
            </div>
            <div className="tag-chip-grid">
              {(im.tags || []).slice(0, 200).map((t) => (
                <span className="tag-chip" key={t}>
                  {t}
                </span>
              ))}
              {(im.tags || []).length > 200 && (
                <span className="muted">
                  … 共 {(im.tags || []).length} 个标签，仅显示前 200 个
                </span>
              )}
              {(im.tags || []).length === 0 && (
                <span className="muted">暂无标签或未同步成功。</span>
              )}
            </div>
          </div>
        ))}
        {(images || []).length === 0 && (
          <div className="empty">尚未添加镜像仓库跟踪。</div>
        )}
      </section>
    </div>
  );
}

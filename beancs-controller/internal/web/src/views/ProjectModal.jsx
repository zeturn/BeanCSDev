import React, { useEffect, useMemo, useRef, useState } from "react";
import {
  Button,
  Input,
  Checkbox,
  Select,
  Textarea,
  Drawer,
  Modal,
} from "../components";
import * as LucideIcons from "lucide-react";
import { envObjectFromEntries, envEntriesFromObject } from "../utils/index";
import { EnvEditor } from "../components/index";
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
export default function ProjectModal({
  project,
  onClose,
  onSubmit,
  onLoadEnv,
}) {
  const [envEntries, setEnvEntries] = useState([]);
  const [envLoading, setEnvLoading] = useState(true);
  const [envError, setEnvError] = useState("");
  useEffect(() => {
    let cancelled = false;
    setEnvLoading(true);
    setEnvError("");
    onLoadEnv(project)
      .then((data) => {
        if (!cancelled) setEnvEntries(envEntriesFromObject(data));
      })
      .catch((err) => {
        if (!cancelled) setEnvError(err.message);
      })
      .finally(() => {
        if (!cancelled) setEnvLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [project.id]);
  const submit = (event) =>
    onSubmit(event, envError ? null : envObjectFromEntries(envEntries));
  return (
    <Modal className="wide-modal">
      <form className="drawer-form" onSubmit={submit}>
        <h2>{t("Edit {name}", { name: project.name })}</h2>
        <label>{t("Display name")}</label>
        <Input name="display_name" defaultValue={project.display_name || ""} />
        <label>{t("Description")}</label>
        <Textarea name="description" defaultValue={project.description || ""} />
        <label>{t("Replicas")}</label>
        <Input
          name="replicas"
          type="number"
          min="0"
          max="20"
          defaultValue={project.replicas || 1}
        />
        <label>{t("Status")}</label>
        <Select name="status" defaultValue={project.status || "active"}>
          <option value="active">{t("Active")}</option>
          <option value="suspended">{t("Suspended")}</option>
          <option value="deleted">{t("Deleted")}</option>
        </Select>
        {project.build_source === "github" && (
          <label className="checkbox-row">
            <Checkbox
              name="auto_deploy"
              type="checkbox"
              defaultChecked={project.auto_deploy !== false}
            />
            {t("Auto build and deploy on GitHub push")}
          </label>
        )}
        {envLoading ? (
          <div className="empty">{t("Loading environment variables...")}</div>
        ) : (
          <>
            {envError && <p className="warning-note">{envError}</p>}
            <EnvEditor
              entries={envEntries}
              onChange={setEnvEntries}
              title={t("Runtime environment")}
              masked
            />
          </>
        )}
        <div className="modal-actions">
          <Button type="button" onClick={onClose}>
            {t("Cancel")}
          </Button>
          <Button variant="primary" type="submit" disabled={envLoading}>
            {t("Save")}
          </Button>
        </div>
      </form>
    </Modal>
  );
}

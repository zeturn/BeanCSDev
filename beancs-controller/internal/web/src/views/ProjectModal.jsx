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
  onLoadVolumes,
  onLoadAvailablePVCs,
}) {
  const [envEntries, setEnvEntries] = useState([]);
  const [envLoading, setEnvLoading] = useState(true);
  const [envError, setEnvError] = useState("");
  const [volumes, setVolumes] = useState([]);
  const [availablePVCs, setAvailablePVCs] = useState([]);
  const [volumesLoading, setVolumesLoading] = useState(true);
  const [volumesError, setVolumesError] = useState("");
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
  useEffect(() => {
    let cancelled = false;
    setVolumesLoading(true);
    setVolumesError("");
    Promise.all([onLoadVolumes(project), onLoadAvailablePVCs(project)])
      .then(([configuredVolumes, claims]) => {
        if (cancelled) return;
        setVolumes(configuredVolumes);
        setAvailablePVCs(claims);
      })
      .catch((err) => {
        if (!cancelled) setVolumesError(err.message);
      })
      .finally(() => {
        if (!cancelled) setVolumesLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [project.id]);
  function updateVolume(index, changes) {
    setVolumes((current) =>
      current.map((volume, currentIndex) =>
        currentIndex === index ? { ...volume, ...changes } : volume,
      ),
    );
  }
  function addVolume() {
    setVolumes((current) => [
      ...current,
      {
        name: "data",
        type: "pvc",
        mountPath: "/data",
        size: "1Gi",
        accessModes: ["ReadWriteOnce"],
      },
    ]);
  }
  const submit = (event) =>
    onSubmit(
      event,
      envError ? null : envObjectFromEntries(envEntries),
      volumes,
    );
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
        <section className="project-volumes">
          <div className="project-volumes-head">
            <h3>
              <HardDrive size={17} /> {t("Volumes")}
            </h3>
            <Button
              type="button"
              variant="icon"
              title={t("Add volume")}
              aria-label={t("Add volume")}
              onClick={addVolume}
              disabled={volumesLoading}
            >
              <Plus size={16} />
            </Button>
          </div>
          {volumesError && <p className="warning-note">{volumesError}</p>}
          {volumesLoading ? (
            <div className="empty">{t("Loading volumes...")}</div>
          ) : (
            <div className="project-volume-list">
              {volumes.map((volume, index) => (
                <div className="project-volume-row" key={`${volume.name}-${index}`}>
                  <Input
                    value={volume.name || ""}
                    placeholder={t("Volume name")}
                    aria-label={t("Volume name")}
                    onChange={(event) => updateVolume(index, { name: event.target.value })}
                  />
                  <Select
                    value={volume.type || "pvc"}
                    aria-label={t("Volume type")}
                    onChange={(event) =>
                      updateVolume(index, {
                        type: event.target.value,
                        ...(event.target.value === "emptyDir"
                          ? { size: "", storageClassName: "", accessModes: [], claimName: "" }
                          : {}),
                      })
                    }
                  >
                    <option value="pvc">{t("New PVC")}</option>
                    <option value="existingPVC">{t("Existing PVC")}</option>
                    <option value="emptyDir">{t("Empty directory")}</option>
                  </Select>
                  <Input
                    value={volume.mountPath || ""}
                    placeholder={t("Mount path")}
                    aria-label={t("Mount path")}
                    onChange={(event) => updateVolume(index, { mountPath: event.target.value })}
                  />
                  {volume.type === "pvc" && (
                    <>
                      <Input
                        value={volume.size || ""}
                        placeholder={t("Size")}
                        aria-label={t("Size")}
                        onChange={(event) => updateVolume(index, { size: event.target.value })}
                      />
                      <Input
                        value={volume.storageClassName || ""}
                        placeholder={t("Storage class")}
                        aria-label={t("Storage class")}
                        onChange={(event) => updateVolume(index, { storageClassName: event.target.value })}
                      />
                      <Select
                        value={volume.accessModes?.[0] || "ReadWriteOnce"}
                        aria-label={t("Access mode")}
                        onChange={(event) => updateVolume(index, { accessModes: [event.target.value] })}
                      >
                        <option value="ReadWriteOnce">ReadWriteOnce</option>
                        <option value="ReadWriteOncePod">ReadWriteOncePod</option>
                        <option value="ReadWriteMany">ReadWriteMany</option>
                        <option value="ReadOnlyMany">ReadOnlyMany</option>
                      </Select>
                    </>
                  )}
                  {volume.type === "existingPVC" && (
                    <Select
                      value={volume.claimName || ""}
                      aria-label={t("Existing PVC")}
                      onChange={(event) => updateVolume(index, { claimName: event.target.value })}
                    >
                      <option value="">{t("Select PVC")}</option>
                      {availablePVCs.map((claim) => (
                        <option key={claim.name} value={claim.name}>
                          {claim.name} ({claim.phase || "Unknown"})
                        </option>
                      ))}
                    </Select>
                  )}
                  <Button
                    type="button"
                    variant="icon"
                    title={t("Remove volume")}
                    aria-label={t("Remove volume")}
                    onClick={() => setVolumes((current) => current.filter((_, currentIndex) => currentIndex !== index))}
                  >
                    <Trash2 size={16} />
                  </Button>
                </div>
              ))}
              {volumes.length === 0 && <div className="empty">{t("No volumes configured.")}</div>}
            </div>
          )}
        </section>
        <div className="modal-actions">
          <Button type="button" onClick={onClose}>
            {t("Cancel")}
          </Button>
          <Button variant="primary" type="submit" disabled={envLoading || volumesLoading}>
            {t("Save")}
          </Button>
        </div>
      </form>
    </Modal>
  );
}

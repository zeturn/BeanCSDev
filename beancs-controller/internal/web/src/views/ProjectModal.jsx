import React, {useEffect, useMemo, useRef, useState} from "react";
import * as LucideIcons from "lucide-react";
import { envObjectFromEntries, envEntriesFromObject } from "../utils/index";
import { EnvEditor } from "../components/index";
import {
  Activity, AlertTriangle, Bell, Boxes, Box, CheckCircle2, ChevronDown, ChevronRight, Cloud, Coffee, Code2, Container, Cpu, Database, Edit3, FileText, GitBranch, Github, Globe2, HardDrive, Image as ImageIcon, KeyRound, Layers3, LayoutDashboard, LineChart, ListRestart, LoaderCircle, Lock, Menu, MemoryStick, MoreHorizontal, Network, Package, Play, Plus, RefreshCw, RotateCcw, Rocket, ScrollText, Search, Server, Settings, Shield, ShieldCheck, Trash2, Upload, X
} from "lucide-react";
export default function ProjectModal({project, onClose, onSubmit, onLoadEnv}) {
  const [envEntries, setEnvEntries] = useState([]);
  const [envLoading, setEnvLoading] = useState(true);
  const [envError, setEnvError] = useState("");
  useEffect(() => {
    let cancelled = false;
    setEnvLoading(true);
    setEnvError("");
    onLoadEnv(project).then((data) => {
      if (!cancelled) setEnvEntries(envEntriesFromObject(data));
    }).catch((err) => {
      if (!cancelled) setEnvError(err.message);
    }).finally(() => {
      if (!cancelled) setEnvLoading(false);
    });
    return () => { cancelled = true; };
  }, [project.id]);
  const submit = (event) => onSubmit(event, envError ? null : envObjectFromEntries(envEntries));
  return (
    <div className="modal-backdrop">
      <form className="modal wide-modal" onSubmit={submit}>
        <h2>Edit {project.name}</h2>
        <label>Display name</label>
        <input name="display_name" defaultValue={project.display_name || ""} />
        <label>Description</label>
        <textarea name="description" defaultValue={project.description || ""} />
        <label>Replicas</label>
        <input name="replicas" type="number" min="0" max="20" defaultValue={project.replicas || 1} />
        <label>Status</label>
        <select name="status" defaultValue={project.status || "active"}>
          <option value="active">Active</option>
          <option value="suspended">Suspended</option>
          <option value="deleted">Deleted</option>
        </select>
        {project.build_source === "github" && (
          <label className="checkbox-row">
            <input name="auto_deploy" type="checkbox" defaultChecked={project.auto_deploy !== false} />
            Auto build and deploy on GitHub push
          </label>
        )}
        {envLoading ? <div className="empty">Loading environment variables...</div> : (
          <>
            {envError && <p className="warning-note">{envError}</p>}
            <EnvEditor entries={envEntries} onChange={setEnvEntries} title="Runtime environment" masked />
          </>
        )}
        <div className="modal-actions">
          <button type="button" onClick={onClose}>Cancel</button>
          <button className="primary" type="submit" disabled={envLoading}>Save</button>
        </div>
      </form>
    </div>
  );
}

import React, {useEffect, useMemo, useRef, useState} from "react";
import * as LucideIcons from "lucide-react";
import { formatTime } from "../utils/index";
import {
  Activity, AlertTriangle, Bell, Boxes, Box, CheckCircle2, ChevronDown, ChevronRight, Cloud, Coffee, Code2, Container, Cpu, Database, Edit3, FileText, GitBranch, Github, Globe2, HardDrive, Image as ImageIcon, KeyRound, Layers3, LayoutDashboard, LineChart, ListRestart, LoaderCircle, Lock, Menu, MemoryStick, MoreHorizontal, Network, Package, Play, Plus, RefreshCw, RotateCcw, Rocket, ScrollText, Search, Server, Settings, Shield, ShieldCheck, Trash2, Upload, X
} from "lucide-react";
export default function WorkloadImageView({images, onRefresh, onOpenRegistry, onRefreshImage, onDeleteImage}) {
  return (
    <div className="stack">
      <section className="panel action-panel">
        <div>
          <h2><ImageIcon size={18} /> Image</h2>
          <p>Running workload images are visible on Pods and Deployments. Tracked registry tags and sync use <b>Integrations → Image Registry</b>.</p>
        </div>
        <button type="button" onClick={onRefresh}><RefreshCw size={15} /> Refresh</button>
      </section>
      <section className="panel">
        <h2><Package size={18} /> Tracked image tags</h2>
        <p className="muted">Mirrors and tag lists you have registered. To add registries or repositories, open Image Registry.</p>
        <div className="row-actions" style={{marginBottom: 12}}>
          <button type="button" className="primary" onClick={onOpenRegistry}><Package size={15} /> Open Image Registry</button>
        </div>
        {(images || []).map((im) => (
          <div className="registry-image-card" key={im.id}>
            <div className="registry-image-head">
              <div>
                <div className="mono strong">{im.repository}</div>
                <small className="muted">{im.registry?.name || `registry #${im.registry_id}`} · {formatTime(im.refreshed_at)}</small>
              </div>
              <div className="row-actions">
                <button type="button" onClick={() => onRefreshImage(im.id)}><RefreshCw size={15} /> Sync</button>
                <button type="button" className="danger-button" onClick={() => onDeleteImage(im)}><Trash2 size={15} /> Remove</button>
              </div>
            </div>
            <div className="tag-chip-grid">
              {(im.tags || []).slice(0, 120).map((t) => (
                <span className="tag-chip" key={t}>{t}</span>
              ))}
              {(im.tags || []).length > 120 && <span className="muted">… {(im.tags || []).length} tags</span>}
              {(im.tags || []).length === 0 && <span className="muted">No tags cached yet.</span>}
            </div>
          </div>
        ))}
        {(images || []).length === 0 && <div className="empty">No tracked images. Configure mirrors under Image Registry.</div>}
      </section>
    </div>
  );
}

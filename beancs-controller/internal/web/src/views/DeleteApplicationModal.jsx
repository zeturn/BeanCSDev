import React, {useEffect, useMemo, useRef, useState} from "react";
import * as LucideIcons from "lucide-react";
import {
  Activity, AlertTriangle, Bell, Boxes, Box, CheckCircle2, ChevronDown, ChevronRight, Cloud, Coffee, Code2, Container, Cpu, Database, Edit3, FileText, GitBranch, Github, Globe2, HardDrive, Image as ImageIcon, KeyRound, Layers3, LayoutDashboard, LineChart, ListRestart, LoaderCircle, Lock, Menu, MemoryStick, MoreHorizontal, Network, Package, Play, Plus, RefreshCw, RotateCcw, Rocket, ScrollText, Search, Server, Settings, Shield, ShieldCheck, Trash2, Upload, X
} from "lucide-react";
import * as Utils from "../utils";
import * as API from "../api";
import * as Components from "../components";
export default function DeleteApplicationModal({application, busy, onClose, onDelete}) {
  return (
    <div className="modal-backdrop">
      <div className="modal">
        <h2>Delete {application.display_name || application.name}</h2>
        <p className="muted">This removes the application record and any managed component/dependency records that are still attached to it.</p>
        <div className="delete-summary">
          <span>Projects <b>{(application.projects || []).length}</b></span>
          <span>Dependencies <b>{(application.dependencies || []).length}</b></span>
          <span>Status <b>{application.status || "-"}</b></span>
        </div>
        <div className="modal-actions">
          <button type="button" onClick={onClose} disabled={busy}>Cancel</button>
          <button className="danger-button filled" type="button" onClick={onDelete} disabled={busy}><Trash2 size={15} /> Delete</button>
        </div>
      </div>
    </div>
  );
}

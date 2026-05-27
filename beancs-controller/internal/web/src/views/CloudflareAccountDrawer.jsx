import React, {useEffect, useMemo, useRef, useState} from "react";
import * as LucideIcons from "lucide-react";
import {
  Activity, AlertTriangle, Bell, Boxes, Box, CheckCircle2, ChevronDown, ChevronRight, Cloud, Coffee, Code2, Container, Cpu, Database, Edit3, FileText, GitBranch, Github, Globe2, HardDrive, Image as ImageIcon, KeyRound, Layers3, LayoutDashboard, LineChart, ListRestart, LoaderCircle, Lock, Menu, MemoryStick, MoreHorizontal, Network, Package, Play, Plus, RefreshCw, RotateCcw, Rocket, ScrollText, Search, Server, Settings, Shield, ShieldCheck, Trash2, Upload, X
} from "lucide-react";
import * as Utils from "../utils";
import * as API from "../api";
import * as Components from "../components";
export default function CloudflareAccountDrawer({onClose, onCreate}) {
  return (
    <div className="side-drawer-backdrop" onClick={onClose}>
      <aside className="side-drawer" onClick={(event) => event.stopPropagation()}>
        <div className="side-drawer-head">
          <div>
            <h2><Cloud size={18} /> Add Cloudflare account</h2>
            <p>Use an API token with zone read access so BeanCS can cache available domains.</p>
          </div>
          <button type="button" className="icon-button" aria-label="Close" onClick={onClose}><X size={16} /></button>
        </div>
        <form className="drawer-form" onSubmit={onCreate}>
          <label>
            Account name
            <input name="name" placeholder="Production Cloudflare" required autoFocus />
          </label>
          <label>
            Account ID
            <input name="account_id" placeholder="Optional, limits zone discovery to this account" />
          </label>
          <label>
            API token
            <input name="api_token" type="password" placeholder="Cloudflare API token" required autoComplete="new-password" />
          </label>
          <div className="drawer-actions">
            <button type="button" onClick={onClose}>Cancel</button>
            <button className="primary" type="submit"><KeyRound size={15} /> Create link</button>
          </div>
        </form>
      </aside>
    </div>
  );
}

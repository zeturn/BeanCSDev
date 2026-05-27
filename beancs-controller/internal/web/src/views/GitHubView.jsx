import React, {useEffect, useMemo, useRef, useState} from "react";
import * as LucideIcons from "lucide-react";
import { GitOpsRepoEditor } from "../components/index";
import {
  Activity, AlertTriangle, Bell, Boxes, Box, CheckCircle2, ChevronDown, ChevronRight, Cloud, Coffee, Code2, Container, Cpu, Database, Edit3, FileText, GitBranch, Github, Globe2, HardDrive, Image as ImageIcon, KeyRound, Layers3, LayoutDashboard, LineChart, ListRestart, LoaderCircle, Lock, Menu, MemoryStick, MoreHorizontal, Network, Package, Play, Plus, RefreshCw, RotateCcw, Rocket, ScrollText, Search, Server, Settings, Shield, ShieldCheck, Trash2, Upload, X
} from "lucide-react";
export default function GitHubView({credentials, onConnect, onUpdate, onRepos, onDelete, reposByCredential, repoFilters, setRepoFilters}) {
  const gitopsRepoRef = useRef(null);
  return (
    <div className="stack">
      <section className="panel action-panel">
        <div>
          <h2><Github size={18} /> GitHub App</h2>
          <p>Authorize repositories directly. BeanCS will name the credential from the GitHub account.</p>
        </div>
        <form onSubmit={(e) => onConnect(e, gitopsRepoRef.current?.value)} style={{display: "flex", gap: "0.5rem", alignItems: "flex-end", flexWrap: "wrap"}}>
          <div style={{display: "flex", flexDirection: "column", gap: "0.25rem"}}>
            <label style={{fontSize: "0.75rem", opacity: 0.7}}>GitOps Repository (optional)</label>
            <input ref={gitopsRepoRef} name="gitops_repo" placeholder="owner/gitops-manifests" style={{minWidth: "240px"}} />
          </div>
          <button className="primary"><Github size={16} /> Connect GitHub App</button>
        </form>
      </section>
      {credentials.map((cred) => {
        const repos = reposByCredential[cred.id] || [];
        const filter = repoFilters[cred.id] || "";
        const visible = repos.filter((repo) => repo.full_name.toLowerCase().includes(filter.toLowerCase()));
        return (
          <section className="panel" key={cred.id}>
            <div className="account-header">
              <div className="account-cell">{cred.avatar_url ? <img src={cred.avatar_url} alt="" /> : <Github size={18} />}<div><b>{cred.name}</b><small>{cred.account_login || cred.org || "-"} · {cred.auth_type || "pat"}</small></div></div>
              <div className="row-actions">
                <button onClick={() => onRepos(cred.id)}><RefreshCw size={15} /> Load repos</button>
                <button onClick={() => onDelete(cred.id)}><Trash2 size={15} /></button>
              </div>
            </div>
            <GitOpsRepoEditor cred={cred} onUpdate={onUpdate} />
            <div className="repo-toolbar">
              <input placeholder="Search repositories" value={filter} onChange={(event) => setRepoFilters((current) => ({...current, [cred.id]: event.target.value}))} />
              <span>{visible.length}/{repos.length} repos</span>
            </div>
            <div className="repo-grid">
              {visible.map((repo) => (
                <a key={repo.full_name} className="repo-card" href={repo.html_url} target="_blank" rel="noreferrer">
                  <b>{repo.full_name}</b>
                  <span>{repo.private ? "Private" : "Public"} · {repo.default_branch || "main"}</span>
                </a>
              ))}
              {repos.length === 0 && <div className="empty">Click Load repos to inspect this account.</div>}
            </div>
          </section>
        );
      })}
    </div>
  );
}

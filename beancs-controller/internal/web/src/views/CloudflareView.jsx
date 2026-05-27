import React, {useEffect, useMemo, useRef, useState} from "react";
import * as LucideIcons from "lucide-react";
import { ExpandableCell } from "../components/index";
import {
  Activity, AlertTriangle, Bell, Boxes, Box, CheckCircle2, ChevronDown, ChevronRight, Cloud, Coffee, Code2, Container, Cpu, Database, Edit3, FileText, GitBranch, Github, Globe2, HardDrive, Image as ImageIcon, KeyRound, Layers3, LayoutDashboard, LineChart, ListRestart, LoaderCircle, Lock, Menu, MemoryStick, MoreHorizontal, Network, Package, Play, Plus, RefreshCw, RotateCcw, Rocket, ScrollText, Search, Server, Settings, Shield, ShieldCheck, Trash2, Upload, X
} from "lucide-react";
export default function CloudflareView({credentials, domains, selectedID, selectedZoneID, setSelectedID, setSelectedZoneID, dnsRecords, editingRecord, setEditingRecord, onCreate, onDelete, onLoadDNS, onSaveDNS, onDeleteDNS}) {
  const [drawerOpen, setDrawerOpen] = useState(false);
  const selected = credentials.find((cred) => String(cred.id) === String(selectedID));
  const selectedDomain = domains.find((domain) => String(domain.credential_id) === String(selectedID) && String(domain.zone_id) === String(selectedZoneID));
  const accountDomains = selected ? domains.filter((domain) => String(domain.credential_id) === String(selected.id)) : [];
  const selectAccount = (cred) => {
    setSelectedID(String(cred.id));
    setSelectedZoneID("");
    setEditingRecord(null);
  };
  const selectDomain = (domain) => {
    setSelectedID(String(domain.credential_id));
    setSelectedZoneID(String(domain.zone_id));
    setEditingRecord(null);
    onLoadDNS(domain.credential_id, domain.zone_id);
  };
  return (
    <div className="stack cloudflare-page">
      <section className="panel action-panel">
        <div>
          <h2><Cloud size={18} /> Cloudflare accounts</h2>
          <p>Link a Cloudflare account once, then choose cached zones from that account.</p>
        </div>
        <button type="button" className="primary" onClick={() => setDrawerOpen(true)}><Plus size={15} /> Add account</button>
      </section>

      <section className="panel">
        <div className="account-header">
          <h2><KeyRound size={18} /> Existing accounts</h2>
          <button type="button" className="danger-button" disabled={!selected} onClick={() => selected && onDelete(selected.id)}><Trash2 size={15} /> Delete selected</button>
        </div>
        <div className="cloudflare-account-grid">
          {credentials.map((cred) => {
            const count = domains.filter((domain) => String(domain.credential_id) === String(cred.id)).length;
            return (
              <button type="button" className={String(selectedID) === String(cred.id) ? "cloudflare-account-card active" : "cloudflare-account-card"} key={cred.id} onClick={() => selectAccount(cred)}>
                <span className="account-mark"><Cloud size={18} /></span>
                <div>
                  <b>{cred.name}</b>
                  <small>{cred.account_id || "No account id"} · {count} domain{count === 1 ? "" : "s"}</small>
                </div>
                <em>{cred.is_active ? "Active" : "Inactive"}</em>
              </button>
            );
          })}
          {credentials.length === 0 && <div className="empty">No Cloudflare accounts linked yet.</div>}
        </div>
      </section>

      <section className="panel">
        <h2><Globe2 size={18} /> {selected ? `${selected.name} domains` : "Account domains"}</h2>
        <div className="domain-grid">
          {accountDomains.map((domain) => (
            <button type="button" className={String(selectedZoneID) === String(domain.zone_id) ? "domain-tile active" : "domain-tile"} key={`${domain.credential_id}-${domain.zone_id}`} onClick={() => selectDomain(domain)}>
              <Globe2 size={20} />
              <div>
                <b>{domain.domain}</b>
                <span>{domain.status || "cached zone"}</span>
                <small>{domain.zone_id}</small>
              </div>
              <em>{domain.active ? "Active" : "Inactive"}</em>
            </button>
          ))}
          {!selected && <div className="empty">Choose a Cloudflare account to view its domains.</div>}
          {selected && accountDomains.length === 0 && <div className="empty">No cached domains for this account.</div>}
        </div>
      </section>

      <section className="panel">
        <div className="account-header">
          <h2><Network size={18} /> DNS records {selectedDomain ? `for ${selectedDomain.domain}` : selected ? `for ${selected.name}` : ""}</h2>
          <button disabled={!selectedID || !selectedZoneID} onClick={() => onLoadDNS(selectedID, selectedZoneID)}><RefreshCw size={15} /> Refresh DNS</button>
        </div>
        <form className="form-grid dns-form" onSubmit={onSaveDNS} key={editingRecord?.id || "new-dns"}>
          <select name="type" defaultValue={editingRecord?.type || "A"}><option>A</option><option>AAAA</option><option>CNAME</option><option>TXT</option><option>MX</option></select>
          <input name="name" placeholder="app.example.com" defaultValue={editingRecord?.name || ""} required />
          <input name="content" placeholder="Target content" defaultValue={editingRecord?.content || ""} required />
          <input name="ttl" type="number" min="1" defaultValue={editingRecord?.ttl || 1} />
          <label className="check-row"><input name="proxied" type="checkbox" defaultChecked={Boolean(editingRecord?.proxied)} /> Proxied</label>
          <input name="comment" placeholder="Comment" defaultValue={editingRecord?.comment || ""} />
          <button className="primary" disabled={!selectedID || !selectedZoneID} type="submit">{editingRecord ? "Save DNS" : "Add DNS"}</button>
          {editingRecord && <button type="button" onClick={() => setEditingRecord(null)}>Cancel</button>}
        </form>
        <div className="table dns-table">
          <div className="tr head"><span>Type</span><span>Name</span><span>Content</span><span>TTL</span><span>Proxy</span><span>Actions</span></div>
          {dnsRecords.map((record) => (
            <div className="tr" key={record.id}>
              <ExpandableCell value={record.type} max={12} /><ExpandableCell value={record.name} max={36} /><ExpandableCell value={record.content} max={42} /><ExpandableCell value={record.ttl} max={12} /><ExpandableCell value={record.proxied ? "Yes" : "No"} max={12} />
              <span className="row-actions"><button onClick={() => setEditingRecord(record)}>Edit</button><button className="danger-button" onClick={() => onDeleteDNS(record)}><Trash2 size={15} /></button></span>
            </div>
          ))}
          {dnsRecords.length === 0 && <div className="empty">{selectedID && selectedZoneID ? "No DNS records loaded." : "Choose a zone to view DNS records."}</div>}
        </div>
      </section>
      {drawerOpen && (
        <CloudflareAccountDrawer
          onClose={() => setDrawerOpen(false)}
          onCreate={async (event) => {
            const ok = await onCreate("cloudflare", event);
            if (ok) setDrawerOpen(false);
          }}
        />
      )}
    </div>
  );
}

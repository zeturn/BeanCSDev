import React, { useEffect, useMemo, useRef, useState } from "react";
import * as LucideIcons from "lucide-react";
import {
  ExpandableCell,
  Button,
  Select,
  Input,
  Checkbox,
} from "../components/index";
import CloudflareAccountDrawer from "./CloudflareAccountDrawer";
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
export default function CloudflareView({
  credentials,
  domains,
  selectedID,
  selectedZoneID,
  setSelectedID,
  setSelectedZoneID,
  dnsRecords,
  editingRecord,
  setEditingRecord,
  onCreate,
  onDelete,
  onLoadDNS,
  onSaveDNS,
  onDeleteDNS,
}) {
  const [drawerOpen, setDrawerOpen] = useState(false);
  const selected = credentials.find(
    (cred) => String(cred.id) === String(selectedID),
  );
  const selectedDomain = domains.find(
    (domain) =>
      String(domain.credential_id) === String(selectedID) &&
      String(domain.zone_id) === String(selectedZoneID),
  );
  const accountDomains = selected
    ? domains.filter(
        (domain) => String(domain.credential_id) === String(selected.id),
      )
    : [];
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
          <h2>
            <Cloud size={18} /> {t("Cloudflare accounts")}
          </h2>
          <p>
            {t(
              "Link a Cloudflare account once, then choose cached zones from that account.",
            )}
          </p>
        </div>
        <Button
          type="button"
          onClick={() => setDrawerOpen(true)}
          variant="primary"
        >
          <Plus size={15} /> {t("Add account")}
        </Button>
      </section>

      <section className="panel">
        <div className="account-header">
          <h2>
            <KeyRound size={18} /> {t("Existing accounts")}
          </h2>
          <Button
            type="button"
            disabled={!selected}
            onClick={() => selected && onDelete(selected.id)}
            variant="danger"
          >
            <Trash2 size={15} /> {t("Delete selected")}
          </Button>
        </div>
        <div className="cloudflare-account-grid">
          {credentials.map((cred) => {
            const count = domains.filter(
              (domain) => String(domain.credential_id) === String(cred.id),
            ).length;
            return (
              <Button
                type="button"
                className={
                  String(selectedID) === String(cred.id)
                    ? "cloudflare-account-card active"
                    : "cloudflare-account-card"
                }
                key={cred.id}
                onClick={() => selectAccount(cred)}
              >
                <span className="account-mark">
                  <Cloud size={18} />
                </span>
                <div>
                  <b>{cred.name}</b>
                  <small>
                    {cred.account_id || t("No account id")} ·{" "}
                    {count}{" "}
                    {count === 1 ? t("domain") : t("domains")}
                  </small>
                </div>
                <em>{cred.is_active ? t("Active") : t("Inactive")}</em>
              </Button>
            );
          })}
          {credentials.length === 0 && (
            <div className="empty">{t("No Cloudflare accounts linked yet.")}</div>
          )}
        </div>
      </section>

      <section className="panel">
        <h2>
          <Globe2 size={18} />{" "}
          {selected
            ? t("{name} domains", { name: selected.name })
            : t("Account domains")}
        </h2>
        <div className="domain-grid">
          {accountDomains.map((domain) => (
            <Button
              type="button"
              className={
                String(selectedZoneID) === String(domain.zone_id)
                  ? "domain-tile active"
                  : "domain-tile"
              }
              key={`${domain.credential_id}-${domain.zone_id}`}
              onClick={() => selectDomain(domain)}
            >
              <Globe2 size={20} />
              <div>
                <b>{domain.domain}</b>
                <span>{domain.status || t("cached zone")}</span>
                <small>{domain.zone_id}</small>
              </div>
              <em>{domain.active ? t("Active") : t("Inactive")}</em>
            </Button>
          ))}
          {!selected && (
            <div className="empty">
              {t("Choose a Cloudflare account to view its domains.")}
            </div>
          )}
          {selected && accountDomains.length === 0 && (
            <div className="empty">{t("No cached domains for this account.")}</div>
          )}
        </div>
      </section>

      <section className="panel">
        <div className="account-header">
          <h2>
            <Network size={18} /> {t("DNS records")}{" "}
            {selectedDomain
              ? t("for {name}", { name: selectedDomain.domain })
              : selected
                ? t("for {name}", { name: selected.name })
                : ""}
          </h2>
          <Button
            disabled={!selectedID || !selectedZoneID}
            onClick={() => onLoadDNS(selectedID, selectedZoneID)}
          >
            <RefreshCw size={15} /> {t("Refresh DNS")}
          </Button>
        </div>
        <form
          className="form-grid dns-form"
          onSubmit={onSaveDNS}
          key={editingRecord?.id || "new-dns"}
        >
          <Select name="type" defaultValue={editingRecord?.type || "A"}>
            <option>A</option>
            <option>AAAA</option>
            <option>CNAME</option>
            <option>TXT</option>
            <option>MX</option>
          </Select>
          <Input
            name="name"
            placeholder="app.example.com"
            defaultValue={editingRecord?.name || ""}
            required
          />
          <Input
            name="content"
            placeholder={t("Target content")}
            defaultValue={editingRecord?.content || ""}
            required
          />
          <Input
            name="ttl"
            type="number"
            min="1"
            defaultValue={editingRecord?.ttl || 1}
          />
          <label className="check-row">
            <Checkbox
              name="proxied"
              type="checkbox"
              defaultChecked={Boolean(editingRecord?.proxied)}
            />{" "}
            {t("Proxied")}
          </label>
          <Input
            name="comment"
            placeholder={t("Comment")}
            defaultValue={editingRecord?.comment || ""}
          />
          <Button
            disabled={!selectedID || !selectedZoneID}
            type="submit"
            variant="primary"
          >
            {editingRecord ? t("Save DNS") : t("Add DNS")}
          </Button>
          {editingRecord && (
            <Button type="button" onClick={() => setEditingRecord(null)}>
              {t("Cancel")}
            </Button>
          )}
        </form>
        <div className="table dns-table">
          <div className="tr head">
            <span>{t("Type")}</span>
            <span>{t("Name")}</span>
            <span>{t("Content")}</span>
            <span>{t("TTL")}</span>
            <span>{t("Proxy")}</span>
            <span>{t("Actions")}</span>
          </div>
          {dnsRecords.map((record) => (
            <div className="tr" key={record.id}>
              <ExpandableCell value={record.type} max={12} />
              <ExpandableCell value={record.name} max={36} />
              <ExpandableCell value={record.content} max={42} />
              <ExpandableCell value={record.ttl} max={12} />
              <ExpandableCell
                value={record.proxied ? t("Yes") : t("No")}
                max={12}
              />
              <span className="row-actions">
                <Button onClick={() => setEditingRecord(record)}>
                  {t("Edit")}
                </Button>
                <Button onClick={() => onDeleteDNS(record)} variant="danger">
                  <Trash2 size={15} />
                </Button>
              </span>
            </div>
          ))}
          {dnsRecords.length === 0 && (
            <div className="empty">
              {selectedID && selectedZoneID
                ? t("No DNS records loaded.")
                : t("Choose a zone to view DNS records.")}
            </div>
          )}
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

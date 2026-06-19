# KorisPanel — Task List

> Updated: 2026-06-19
> Completed / verified items removed. Remaining work for v1.0 release.

---

## ✅ Completed This Session

- All 6 critical bugs fixed
- All 16 verification bugs confirmed
- SSL/HTTPS auto-detection + nginx management API + TLS fallback
- Portal login error display fixed
- Cores tab redesigned (WireGuard uniform, SSH status fixed, auto-start/stop)
- Backup moved to Settings sub-tab
- Node agent: cross-platform build, config management from panel
- Custom config filenames (emoji/UTF-8, per-user vs global logic)
- Dual OpenVPN profiles (UDP + TCP) with failover
- Per-user preferred node selection (portal API + UI)
- Static configs with global vpn_domain + backup domains
- Performance: indexes, /20 subnets, 2-core/4GB tuning, MariaDB optimization
- Promo codes (backend + admin UI + portal UI)
- Grace period enforcement
- Multi-currency support (DB schema)
- Passwordless configs
- WireGuard peers automated (customer detail view)

---

## ⭐ New Features — Medium Priority

- [ ] **Cisco IPSec Protocol**
  - Add Cisco IPSec VPN server support
  - Enterprise client compatibility

- [ ] **Drag & Drop Reordering**
  - Remove "sort order" from all lists
  - Add drag & drop for items

- [ ] **Package Distribution**
  - Build .deb package for panel
  - Build .deb/.rpm package for node

- [ ] **Windows Node Support**
  - Run node agent on Windows Server

---

## 🟢 Enhancements

- [ ] **Ticket System Improvements**
  - Show "Create Ticket" button by default
  - Add notification on new/updated tickets
  - Add delete/archive action

- [ ] **Balance System Enhancement**
  - Major improvements to wallet/credit system
  - Better flexibility and limits
  - Improved transaction tracking

- [ ] **Changelog System**
  - Full changelog with every change logged
  - Auto-update on each PR/commit, version history

---

## 🔵 Infrastructure — Network Features

- [ ] **QoS / Priority System**
  - tc + fwmark for traffic shaping
  - Per-user bandwidth priority

- [ ] **Firewall Module**
  - iptables/nftables integration
  - Country blocking, rate limits

- [ ] **Layer 7 Filtering**
  - P2P detection/blocking

- [ ] **Load Balancing**
  - Multi-WAN, automatic failover

- [ ] **Bandwidth Control**
  - Per-user PCQ/HTB shaping via tc

---

## 💰 Business & Billing

- [x] ~~Promo Codes~~ — backend + admin UI + portal UI
- [x] ~~Grace Period~~ — per-plan grace_days, worker enforcement
- [x] ~~Multi-Currency~~ — DB schema (plans.currency, toman_rate)
- [ ] **Referral System** — credit on referral, commission, tracking
- [ ] **Multi-Config Purchase** — buy extra connection slots
- [ ] **Connection Limit Per User** — admin override per user

---

## 🔐 Security Features

- [ ] **Admin Roles / Permissions** — RBAC, granular permissions
- [ ] **Activity Log / Audit Trail** — searchable, exportable (partially exists)
- [ ] **Session Management** — active sessions, force logout, IP tracking

---

## 📱 Client Portal Enhancements

- [x] ~~Node selection~~ — dropdown in portal, preferred node saved
- [x] ~~Promo code input~~ — apply codes in portal
- [ ] **Usage Notifications** — 80%/90%/100% alerts via Telegram
- [ ] **Auto-Renewal** — charge from wallet
- [ ] **Mobile Responsive** — better mobile UI
- [ ] **Timezone Per User** — auto-detect, local time

---

## 📊 Reporting & Analytics

- [ ] **Revenue Reports** — daily/weekly/monthly
- [ ] **User Reports** — registrations, retention
- [ ] **Bandwidth Reports** — per-node, per-user
- [ ] **Export PDF / Excel** — downloadable reports

---

## 🛠️ Technical Features

- [ ] **Webhooks** — event notifications
- [ ] **Trial Period** — configurable days, auto-convert
- [ ] **Time-Based Plans** — hourly, daily, weekly
- [ ] **Data Packs** — buy extra data
- [ ] **Plan Upgrade / Downgrade** — pro-rated

---

## 🚨 Monitoring & Alerts

- [ ] **Alert System** — Telegram alerts on events
- [ ] **Uptime Monitoring** — node health checks
- [ ] **Server Maintenance** — scheduled mode

---

## 🧪 Testing & Docs

- [ ] Test all protocols
- [ ] Performance testing (500+ users)
- [ ] Security audit
- [ ] Admin documentation
- [ ] API documentation
- [ ] Installation guide

---

## 🔮 Future (Not Priority)

- Xray/VLESS protocol (smart TCP proxy per-user routing)
- HAProxy integration for TCP OpenVPN node selection
- Anti-DPI Integration
- LDAP/AD Integration
- Server Clustering
- RTL Support
- Package distribution (.deb/.rpm)

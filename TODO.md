# KorisPanel — Task List

> Updated: 2026-06-18
> Completed items removed. Remaining work for v1.0 release.

---

## 🟡 Bugs to Verify (Previously Fixed)

- [ ] Unnecessary session disconnection — disconnect on unchanged fields
- [ ] Vite version conflict (ERESOLVE) — reverted to Vite 5.x
- [ ] WireGuard AllowedIPs validation — net.ParseCIDR in createWireguardPeer
- [ ] WireGuard config sync fix — wg-quick strip piped to temp file
- [ ] WireGuard remove peer trailing newline — config file corruption
- [ ] OpenVPN empty network check — explicit check matching old behavior
- [ ] Proxy persistence — per-node API proxy
- [ ] SSH status fallback — checks status_metrics.ssh_status
- [ ] L2TP redundant toggles — removed refuse_chap/refuse_pap/require_mschapv2

---

## ⭐ New Features — High Priority

- [ ] **WireGuard Protocol Support**
  - Add WireGuard VPN server support on nodes
  - Include gaming optimize option
  - Low latency, high throughput

- [ ] **Passwordless Configs**
  - Default: configs include username/password
  - Optional: users can generate passwordless configs
  - Admin setting: enable/disable globally
  - Per-plan setting: allow passwordless for specific plans
  - Protocols: OpenVPN, L2TP, IKEv2, WireGuard
  - Generate config without auth-user-pass line
  - Certificate-based or pre-shared key auth

- [ ] **Tunnel Mode (Iran Traffic)**
  - Users connect to Iran node
  - Traffic forwarded to outbound server
  - Essential for Iran market

- [ ] **Backup System Upgrade**
  - Replace JSON export with SQL dump
  - Proper backup/restore functionality
  - Database + configs package

---

## ⭐ New Features — Medium Priority

- [ ] **Cisco IPSec Protocol**
  - Add Cisco IPSec VPN server support
  - Enterprise client compatibility

- [ ] **Gaming Optimize Option**
  - Per-user speed boost for gaming
  - Low latency mode, bandwidth priority

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

- [ ] **Settings Section Redesign**
  - Complete redesign of settings page
  - Better organization, improved UX/UI

- [ ] **Cores Settings Enhancement**
  - Current implementation inadequate
  - More control options

- [ ] **Changelog System**
  - Full changelog with every change logged
  - Auto-update on each PR/commit, version history

---

## 🔵 Infrastructure — Network Features

- [ ] **QoS / Priority System**
  - tc + fwmark for traffic shaping
  - Per-user bandwidth priority
  - Gaming/VoIP optimization
  - Deps: iproute2, iptables

- [ ] **Firewall Module**
  - iptables/nftables integration
  - Advanced filtering, country blocking, rate limits

- [ ] **Layer 7 Filtering**
  - Protocol-based traffic filtering
  - P2P detection/blocking
  - Deps: iptables-mod-layer7

- [ ] **Load Balancing**
  - Multi-WAN support, PCC-like config
  - Automatic failover, round-robin / sticky sessions

- [ ] **Bandwidth Control**
  - Per-user PCQ/HTB shaping via tc
  - MikroTik-like queue system
  - Real-time speed management

---

## 💰 Business & Billing

- [ ] **Payment Gateway**
  - Stripe, PayPal, Crypto (USDT, BTC)
  - Automatic payment verification

- [ ] **Local Iran Payments**
  - Shetab card, Mellat gateway, ZarinPal

- [ ] **Promo Codes / Coupons**
  - Percentage or fixed amount, usage limits, expiry dates

- [ ] **Referral System**
  - Credit on referral signup, commission percentage, tracking

- [ ] **Multi-Currency**
  - Toman, USD, EUR
  - Automatic conversion, per-plan currency

- [ ] **Grace Period**
  - Extended access after expiry
  - Configurable days, partial access, notification

- [ ] **Multi-Config Purchase (Multi-Login)**
  - Buy additional connection slots
  - Default connections per plan, price per extra
  - Admin override per user

- [ ] **Connection Limit Per User**
  - Admin custom limit override
  - Temporary increase/decrease, audit trail

---

## 🔐 Security Features

- [ ] **Admin Roles / Permissions**
  - Multi-admin, RBAC, granular permissions

- [ ] **Activity Log / Audit Trail**
  - Log all admin actions, searchable, exportable

- [ ] **Session Management**
  - Active sessions list, force logout, timeout settings, IP tracking

---

## 📱 Client Portal Enhancements

- [ ] **Usage Notifications**
  - Alert at 80% / 90% / 100% data limit
  - Email / SMS / Telegram alerts

- [ ] **Auto-Renewal**
  - Charge from wallet, email confirmation, configurable

- [ ] **Mobile Responsive**
  - Better mobile UI/UX, touch-friendly, mobile-optimized tables

- [ ] **Timezone Per User**
  - Auto-detect, local time display

---

## 📊 Reporting & Analytics

- [ ] **Revenue Reports** — daily/weekly/monthly, by plan, by method
- [ ] **User Reports** — registrations, active vs churned, retention
- [ ] **Bandwidth Reports** — per-node, per-user, peak times
- [ ] **Profit / Loss** — costs vs revenue, margin per plan
- [ ] **Export PDF / Excel** — downloadable, scheduled
- [ ] **Custom Dashboard** — drag & drop widgets, role-specific

---

## 🛠️ Technical Features

- [ ] **Webhooks** — event notifications (user created, payment, expiry)
- [ ] **Auto Backup** — scheduled, cloud (S3), local, rotation
- [ ] **Trial Period** — configurable days, limited features, auto-convert
- [ ] **Time-Based Plans** — hourly, daily, weekly, pay-per-use
- [ ] **Data Packs** — buy extra data, add-on packages, stack with plan
- [ ] **Plan Upgrade / Downgrade** — mid-cycle, pro-rated, change history

---

## 🚨 Monitoring & Alerts

- [ ] **Alert System** — email, Telegram, SMS, custom rules
- [ ] **Uptime Monitoring** — health checks, response time, alert on downtime
- [ ] **Server Maintenance** — scheduled mode, user notification, countdown

---

## 🧪 Testing

- [ ] Test all protocols (OpenVPN, L2TP, IKEv2, SSH, WireGuard)
- [ ] Performance testing (500+ users)
- [ ] Security audit
- [ ] Migration guide

---

## 📖 Documentation

- [ ] Complete admin documentation
- [ ] Complete API documentation
- [ ] Complete user guide
- [ ] Complete installation guide

---

## 🔮 Future (Not Priority)

- Anti-DPI Integration
- LDAP/AD Integration
- Server Clustering
- Auto Server Switch
- RTL Support

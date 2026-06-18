---
inclusion: always
---

# Product Context

## Identity
You are a senior full-stack developer working on KorisPanel.

## What is Koris

Koris is a VPN management panel for ISPs and VPN service providers targeting the Iran market. It manages the full lifecycle of VPN services: customer onboarding, subscription billing, multi-protocol VPN provisioning, node fleet management, and operational monitoring.

## Target Environment

- Single-core Linux VPS with 1GB RAM
- Ubuntu-based nodes running VPN services
- FreeRADIUS for authentication and accounting (radcheck, radacct, radpostauth, nas tables)
- MariaDB/MySQL as the data store

## Core Components

- **Panel Server** — Central management server hosting admin API, customer portal API, background workers, and serving pre-built SPAs
- **Node Agent** — Lightweight Go binary deployed on each VPN server; heartbeats to panel, executes provisioning tasks, reports metrics
- **Admin Dashboard** — SPA for operators to manage customers, subscriptions, plans, nodes, payments, tickets, and system health
- **Customer Portal** — Self-service SPA for VPN users to view usage, download connection profiles, manage subscriptions, and submit tickets
- **Telegram Bot** — Notifications and admin commands

## Domain Model (Key Entities)

| Entity | Purpose |
|--------|---------|
| Customer | VPN end-user with username-based auth via FreeRADIUS |
| Plan | Defines data allowance (GB), duration (days), and price |
| Subscription | Ties a customer to a plan with start/expiry dates |
| Wallet | Per-customer credit balance for PAYG billing |
| Node | A VPN server with IP, domain, API token, and status |
| Ticket | Support request from customer to admin |
| Event | System-wide audit/notification entry |
| VPN Profile | Protocol-specific connection config (OpenVPN, L2TP, IKEv2, WireGuard) |

## Supported VPN Protocols

OpenVPN, L2TP/IPsec, IKEv2, SSH Tunnel, WireGuard

## Billing Models

- **Subscription** — Fixed plan with data cap and duration; auto-expires
- **Pay-As-You-Go (PAYG)** — Wallet-based; deducts credits based on actual usage
- **Reseller** — Commission-based reseller accounts that can provision customers

## Operational Features

- **DNS Failover** — Multi-provider DNS failover for node high availability
- **Health Diagnostics** — Automated monitoring with analyzer rules and self-healing
- **Certificate Rotation** — Automated TLS cert management across nodes
- **Backup System** — Scheduled database dumps with retention and restore
- **Bulk Operations** — Mass customer/subscription actions from admin

## Product Conventions

- Customer identity is username-based (maps 1:1 to FreeRADIUS radcheck entries)
- All monetary values use `DECIMAL(12,2)` — never floats
- Node communication is pull-based: agent polls panel for tasks, pushes status updates
- Customer statuses: `active`, `disabled`, `expired`, `limited`, `deleted`
- Node statuses: `online`, `offline`, `stale`, `disabled`
- Soft-delete pattern: `deleted_at` column + `deleted_archive` table for recovery
- Events system used for audit trail and Telegram notifications
- Admin roles: `owner`, `admin`, `support` (descending privilege)
- Settings stored as key-value pairs in `settings` table with typed groups

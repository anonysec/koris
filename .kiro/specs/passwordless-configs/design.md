# Design Document: Passwordless VPN Configs

## Overview

This design implements optional passwordless VPN configuration generation for KorisPanel. Currently all VPN profiles require username/password authentication (auth-user-pass for OpenVPN, user credentials for L2TP/IKEv2). This feature adds certificate-based and PSK-only authentication as an alternative, controlled by a global admin toggle and per-plan permission flags.

### Key Design Decisions

1. **New internal package**: A `panel/internal/passwordless/` package encapsulates certificate management, config rendering, and revocation logic to keep `api.go` focused on routing.
2. **Existing panel_settings pattern**: The global toggle and expiry setting use the established `panel_settings` key-value table, consistent with how all other global settings work.
3. **Plans table extension**: A simple TINYINT column on the plans table controls per-plan access. No new junction table needed.
4. **Certificate storage with encryption**: Client private keys are encrypted at rest using AES-256-GCM with a server-side key. This follows the same pattern as WireGuard private_key_encrypted storage.
5. **Query parameter for mode switching**: The existing `/api/portal/profiles/` endpoint accepts `?passwordless=1` to serve passwordless configs, avoiding new endpoint proliferation.
6. **WireGuard excluded**: WireGuard is already key-based by nature. No code changes for WireGuard profiles.

## Architecture

```
Customer Portal                Panel Backend                    Database
     │                              │                              │
     │  GET /profiles/openvpn.ovpn  │                              │
     │    ?passwordless=1           │                              │
     │─────────────────────────────>│                              │
     │                              │  Check global toggle         │
     │                              │─────────────────────────────>│
     │                              │  Check plan permission       │
     │                              │─────────────────────────────>│
     │                              │  GetOrCreateCert             │
     │                              │─────────────────────────────>│
     │                              │  Render passwordless config  │
     │<─────────────────────────────│                              │
     │  .ovpn with cert/key         │                              │
```

```
┌─────────────────────────────────────────────────────────┐
│  panel/internal/api/                                     │
│  ┌──────────────┐  ┌──────────────┐  ┌───────────────┐ │
│  │ panelSettings│  │ plans CRUD   │  │ portalProfile  │ │
│  │ (toggle)     │  │ (flag)       │  │ Download       │ │
│  └──────────────┘  └──────────────┘  └───────┬───────┘ │
│                                               │         │
│  ┌────────────────────────────────────────────▼───────┐ │
│  │ panel/internal/passwordless/                        │ │
│  │  ┌─────────────┐  ┌────────────┐  ┌────────────┐  │ │
│  │  │ certmanager │  │ openvpn    │  │ mobileconf │  │ │
│  │  │ (generate,  │  │ (render    │  │ (l2tp,     │  │ │
│  │  │  revoke,    │  │  .ovpn)    │  │  ikev2)    │  │ │
│  │  │  store)     │  │            │  │            │  │ │
│  │  └─────────────┘  └────────────┘  └────────────┘  │ │
│  └────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────┘
```

## Components and Interfaces

### 1. Panel API Endpoints

#### Modified Admin Endpoints

| Method | Path | Change |
|--------|------|--------|
| GET/PATCH | `/api/panel-settings` | No change (already generic key-value); frontend reads/writes `allow_passwordless_configs` and `passwordless_config_expiry_days` |
| GET/POST/PATCH | `/api/plans`, `/api/plans/{id}` | Add `allow_passwordless` field to Plan struct and queries |
| POST | `/api/customers/{id}/revoke-cert` | **New** — Revokes active client certificate for a customer |

#### Modified Portal Endpoints

| Method | Path | Change |
|--------|------|--------|
| GET | `/api/portal/profiles` | Add `passwordless_available` boolean to each profile in response |
| GET | `/api/portal/profiles/{proto}` | Accept `?passwordless=1` query param; return cert-based or PSK-only config |

### 2. New Package: `panel/internal/passwordless/`

#### certmanager.go

```go
type CertManager struct {
    DB        *sql.DB
    CAKeyPath string  // /etc/openvpn/server/ca.key or env override
    CACrtPath string  // /etc/openvpn/server/ca.crt or env override
    EncKey    []byte  // AES-256 key from PANEL_CERT_ENCRYPTION_KEY env
}

func (cm *CertManager) GetOrCreateCert(customerID int64, protocol string, expiryDays int) (*ClientCert, error)
func (cm *CertManager) RevokeCert(certID int64) error
func (cm *CertManager) RegenerateCRL() ([]byte, error)
func (cm *CertManager) CleanupExpired() (int64, error)
```

#### openvpn.go

```go
type OVPNParams struct {
    Username, NodeName, Host string
    Port                     int
    Proto                    string
    CACert, TLSCryptKey      string // PEM content
    ClientCert, ClientKey    string // PEM content
}

func RenderPasswordlessOVPN(params OVPNParams) (string, error)
```

#### mobileconfig.go

```go
func RenderPasswordlessL2TP(host, psk, identifier string) (string, error)
func RenderPasswordlessIKEv2(host string, certP12 []byte, certPasswd, identifier string) (string, error)
```

### 3. Node Agent Extension

| Task Action | Payload | Behavior |
|-------------|---------|----------|
| `crl.update` | `{crl_pem: "..."}` | Write CRL to `/etc/openvpn/server/crl.pem`, reload OpenVPN |

The OpenVPN server template needs `crl-verify /etc/openvpn/server/crl.pem` directive added.

### 4. Frontend Components

#### Admin UI

- Settings → General: "Passwordless Configs" section with toggle + expiry input
- Plan form: "Allow Passwordless Configs" checkbox (disabled when global toggle off)
- Customer detail: "Revoke Passwordless Certificate" button

#### Portal UI

- `ConfigTypeSelector.vue`: Standard/Passwordless segmented toggle per profile
- Hidden for WireGuard (inherently key-based)
- Visible only when `passwordless_available: true`

## Data Models

### Migration: `XXX_passwordless_configs.sql`

```sql
-- Per-plan passwordless permission
ALTER TABLE plans ADD COLUMN allow_passwordless TINYINT(1) NOT NULL DEFAULT 0;

-- Client certificates table
CREATE TABLE client_certificates (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    customer_id BIGINT UNSIGNED NOT NULL,
    protocol ENUM('openvpn', 'ikev2') NOT NULL,
    private_key_encrypted TEXT NOT NULL,
    certificate_pem TEXT NOT NULL,
    serial_number VARCHAR(64) NOT NULL,
    status ENUM('active', 'revoked', 'expired') NOT NULL DEFAULT 'active',
    issued_at DATETIME NOT NULL,
    expires_at DATETIME NOT NULL,
    revoked_at DATETIME DEFAULT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_customer_protocol_status (customer_id, protocol, status),
    INDEX idx_serial (serial_number)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Default panel settings for passwordless feature
INSERT INTO panel_settings (setting_key, setting_value)
VALUES ('allow_passwordless_configs', '0'),
       ('passwordless_config_expiry_days', '30')
ON DUPLICATE KEY UPDATE setting_key = setting_key;
```

### Plans Table Extension

| Column | Type | Default | Description |
|--------|------|---------|-------------|
| allow_passwordless | TINYINT(1) | 0 | Whether customers on this plan can download passwordless configs |

### Client Certificates Table

| Column | Type | Description |
|--------|------|-------------|
| id | BIGINT UNSIGNED | Auto-increment primary key |
| customer_id | BIGINT UNSIGNED | References customers table |
| protocol | ENUM('openvpn','ikev2') | Which protocol cert is for |
| private_key_encrypted | TEXT | AES-256-GCM encrypted client private key |
| certificate_pem | TEXT | X.509 certificate in PEM format |
| serial_number | VARCHAR(64) | Unique certificate serial (hex) |
| status | ENUM | active, revoked, or expired |
| issued_at | DATETIME | Certificate issue timestamp |
| expires_at | DATETIME | Certificate expiry timestamp |
| revoked_at | DATETIME | Revocation timestamp (NULL if not revoked) |

### Generated Config Formats

**OpenVPN Passwordless:**
```
client
dev tun
proto udp
remote host port
# ... standard directives ...
# NO auth-user-pass
verify-x509-name server name
<ca>
... CA cert ...
</ca>
<cert>
... client cert ...
</cert>
<key>
... client key ...
</key>
<tls-crypt>
... tls-crypt key ...
</tls-crypt>
```

**L2TP Passwordless (.mobileconfig):**
```xml
<dict>
    <key>AuthenticationMethod</key>
    <string>SharedSecret</string>
    <key>SharedSecret</key>
    <string>{PSK}</string>
    <!-- No XAuth username/password -->
</dict>
```

**IKEv2 Passwordless (.mobileconfig):**
```xml
<dict>
    <key>AuthenticationMethod</key>
    <string>Certificate</string>
    <key>PayloadCertificateUUID</key>
    <string>{cert-uuid}</string>
</dict>
```

## Correctness Properties

*A property is a characteristic or behavior that should hold true across all valid executions of a system — essentially, a formal statement about what the system should do.*

### Property 1: Standard config unchanged when passwordless not requested

*For any* valid customer with an active subscription, requesting a config download without `?passwordless=1` SHALL produce output identical to the current implementation (contains `auth-user-pass` for OpenVPN, contains username/password fields for L2TP/IKEv2).

**Validates: Requirements 7.2, 11.2**

### Property 2: Passwordless OpenVPN structural invariant

*For any* valid passwordless OpenVPN config generation, the output SHALL contain `<cert>`, `<key>`, and `<ca>` inline blocks AND SHALL NOT contain the string `auth-user-pass`.

**Validates: Requirements 3.1, 3.2, 3.5**

### Property 3: Standard OpenVPN structural invariant

*For any* standard (non-passwordless) OpenVPN config generation, the output SHALL contain `auth-user-pass` AND SHALL NOT contain `<cert>` or `<key>` inline blocks.

**Validates: Requirements 3.5, 7.2**

### Property 4: Permission gate enforcement

*For any* passwordless config request, if the global toggle `allow_passwordless_configs` is "0" OR the customer's plan has `allow_passwordless = 0`, the response SHALL be HTTP 403.

**Validates: Requirements 1.5, 2.5, 11.3, 11.4**

### Property 5: Certificate uniqueness per customer per protocol

*For any* customer and protocol pair, there SHALL be at most one certificate with status "active" in the client_certificates table at any time.

**Validates: Requirements 9.3**

### Property 6: Certificate expiry correctness

*For any* generated certificate, the `expires_at` timestamp SHALL equal `issued_at` plus the configured `passwordless_config_expiry_days` value (converted to days).

**Validates: Requirements 8.2, 8.3**

### Property 7: OpenVPN config round-trip

*For any* valid set of PEM inputs (CA cert, client cert, client key, tls-crypt key), generating a passwordless .ovpn config and extracting the inline blocks SHALL yield PEM content identical to the original inputs.

**Validates: Requirements 3.5**

## Error Handling

| Scenario | Response | Recovery |
|----------|----------|----------|
| Passwordless disabled globally | 403 `{"error": "passwordless_disabled"}` | Admin enables in settings |
| Plan does not allow passwordless | 403 `{"error": "plan_passwordless_not_allowed"}` | Admin enables on plan |
| CA key/cert not found on filesystem | 500 `{"error": "ca_not_configured"}` | Admin sets up CA |
| PANEL_CERT_ENCRYPTION_KEY not set | Startup warning, feature disabled | Set env var |
| L2TP PSK not configured | 400 `{"error": "psk_not_configured"}` | Admin configures IPSec PSK |
| Certificate generation failure | 500 `{"error": "cert_generation_failed"}` | Check CA files and permissions |
| Customer has no active plan | 403 `{"error": "no_active_plan"}` | Customer renews subscription |
| CRL push to node fails | Task status "failed" | Admin retries or checks node |

## Testing Strategy

### Property-Based Tests

Property-based testing is appropriate here because the core functions (config rendering, permission checking, certificate generation) are pure or near-pure functions with clear input/output contracts.

**Library:** `pgregory.net/rapid` for Go backend, `fast-check` for frontend (TypeScript)

**Configuration:** Minimum 100 iterations per property test.

| Property | Test Location | What Varies |
|----------|---------------|-------------|
| P1: Standard unchanged | `panel/internal/passwordless/openvpn_test.go` | Random usernames, hosts, ports, cert content |
| P2: Passwordless structural | `panel/internal/passwordless/openvpn_test.go` | Random valid PEM blocks |
| P3: Standard structural | `panel/internal/passwordless/openvpn_test.go` | Random usernames, hosts |
| P4: Permission gate | `panel/internal/api/api_passwordless_test.go` | Random toggle/plan flag combinations |
| P5: Cert uniqueness | `panel/internal/passwordless/certmanager_test.go` | Multiple concurrent cert requests |
| P6: Expiry correctness | `panel/internal/passwordless/certmanager_test.go` | Random expiry day values |
| P7: Config round-trip | `panel/internal/passwordless/openvpn_test.go` | Random valid PEM content |

### Unit Tests (example-based)

- Certificate generation produces valid X.509 cert
- AES encryption/decryption round-trip for private keys
- L2TP .mobileconfig is valid XML with correct payload structure
- IKEv2 .mobileconfig references certificate UUID correctly
- PKCS#12 bundle creation from cert + key
- Permission helper functions with various DB states

### Integration Tests

- Full flow: enable globally → enable on plan → download passwordless → verify output
- Revoke cert → regenerate CRL → new download creates fresh cert
- Standard config regression: verify unchanged output
- Expiry worker marks expired certs

### Smoke Tests

- Migration applies cleanly
- Panel starts with PANEL_CERT_ENCRYPTION_KEY set
- Settings page loads with new fields

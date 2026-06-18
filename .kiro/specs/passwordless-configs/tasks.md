# Implementation Plan: Passwordless VPN Configs

## Overview

This plan implements optional passwordless VPN configuration generation for KorisPanel. The feature adds certificate-based (OpenVPN, IKEv2) and PSK-only (L2TP) authentication as alternatives to username/password credentials. Implementation covers: database migration, certificate management package, config renderers, API modifications, CRL distribution, admin UI settings, plan form updates, and customer portal config type selection.

## Tasks

- [ ] 1. Database schema and backend foundation
  - [ ] 1.1 Create migration file `panel/migrations/XXX_passwordless_configs.sql`
    - ALTER TABLE plans ADD COLUMN allow_passwordless TINYINT(1) NOT NULL DEFAULT 0
    - CREATE TABLE client_certificates with columns: id, customer_id, protocol, private_key_encrypted, certificate_pem, serial_number, status, issued_at, expires_at, revoked_at, created_at
    - Add indexes on (customer_id, protocol, status) and (serial_number)
    - INSERT default panel_settings entries: allow_passwordless_configs=0, passwordless_config_expiry_days=30
    - _Requirements: 12.1, 12.2, 12.3, 12.4_

  - [ ] 1.2 Update Plan struct and CRUD queries in `panel/internal/api/api.go`
    - Add `AllowPasswordless bool` field with json tag `allow_passwordless` to Plan struct
    - Update scanPlan to scan allow_passwordless column
    - Update listPlans, createPlan, updatePlan, getPlan queries to include allow_passwordless
    - Update backup export/import to include allow_passwordless column
    - _Requirements: 2.1, 2.2, 2.3, 2.4_

  - [ ] 1.3 Add passwordless permission helper functions in `panel/internal/api/api.go`
    - Implement `isPasswordlessEnabled(db) bool` — reads allow_passwordless_configs from panel_settings
    - Implement `planAllowsPasswordless(db, username) bool` — checks customer's plan allow_passwordless flag
    - Implement `getPasswordlessExpiryDays(db) int` — reads passwordless_config_expiry_days, defaults to 30
    - _Requirements: 1.4, 2.4, 8.2_

- [ ] 2. Certificate manager package
  - [ ] 2.1 Create `panel/internal/passwordless/` package with `certmanager.go`
    - Define CertManager struct with DB, CAKeyPath, CACrtPath, EncKey fields
    - Define ClientCert struct (ID, CustomerID, Protocol, CertPEM, KeyPEM, SerialNumber, Status, IssuedAt, ExpiresAt)
    - _Requirements: 9.1_

  - [ ] 2.2 Implement certificate generation logic
    - Generate RSA 2048-bit key pair
    - Create X.509 certificate signed by panel CA with configurable expiry
    - Set Subject CN to customer username, serial number from crypto/rand
    - Return cert PEM and key PEM
    - _Requirements: 9.1, 8.3_

  - [ ] 2.3 Implement private key encryption/decryption
    - Encrypt with AES-256-GCM using PANEL_CERT_ENCRYPTION_KEY env var
    - Decrypt on config download
    - _Requirements: 9.2_

  - [ ] 2.4 Implement GetOrCreateCert(customerID, protocol, expiryDays)
    - Query for active cert: SELECT FROM client_certificates WHERE customer_id=? AND protocol=? AND status='active' AND expires_at > NOW()
    - If found, decrypt and return
    - If not found or expired, generate new cert, encrypt key, INSERT, return
    - Mark old expired cert as status='expired' if exists
    - _Requirements: 9.3, 9.5_

  - [ ] 2.5 Implement RevokeCert(certID)
    - UPDATE client_certificates SET status='revoked', revoked_at=NOW() WHERE id=?
    - _Requirements: 10.2_

  - [ ] 2.6 Implement RegenerateCRL()
    - Query all revoked certs, build X.509 CRL signed by CA, return PEM bytes
    - _Requirements: 10.3_

  - [ ] 2.7 Implement CleanupExpired()
    - UPDATE client_certificates SET status='expired' WHERE status='active' AND expires_at < NOW()
    - Return count of affected rows
    - _Requirements: 9.5_

  - [ ] 2.8 Write unit tests for certmanager with sqlmock
    - Test GetOrCreateCert returns existing active cert
    - Test GetOrCreateCert generates new cert when none exists
    - Test RevokeCert updates status correctly
    - Test encryption/decryption round-trip
    - _Requirements: 9.1, 9.2, 9.3_

- [ ] 3. Passwordless config renderers
  - [ ] 3.1 Create `panel/internal/passwordless/openvpn.go` with RenderPasswordlessOVPN
    - Accept OVPNParams: Username, NodeName, Host, Port, Proto, CACert, TLSCryptKey, ClientCert, ClientKey
    - Generate .ovpn content: standard directives, NO auth-user-pass, inline <ca>, <cert>, <key>, <tls-crypt> blocks
    - Add verify-x509-name directive for server cert validation
    - _Requirements: 3.1, 3.2, 3.3_

  - [ ] 3.2 Create `panel/internal/passwordless/mobileconfig.go` with RenderPasswordlessL2TP
    - Generate .mobileconfig XML with VPN payload type L2TP
    - Set AuthenticationMethod=SharedSecret, embed PSK
    - No XAuth/username/password fields
    - Return error if PSK is empty
    - _Requirements: 4.1, 4.2, 4.3, 4.4_

  - [ ] 3.3 Add RenderPasswordlessIKEv2 to mobileconfig.go
    - Generate .mobileconfig XML with VPN payload type IKEv2
    - Set AuthenticationMethod=Certificate
    - Create PKCS#12 bundle from client cert + key
    - Embed PKCS#12 as separate certificate payload with UUID reference
    - _Requirements: 5.1, 5.2, 5.3_

  - [ ]* 3.4 Write property test: Passwordless OpenVPN structural invariant (Property 2)
    - **Property 2: Passwordless OpenVPN structural invariant**
    - Generate random valid PEM blocks as inputs; verify output contains <cert>, <key>, <ca> and does NOT contain auth-user-pass
    - Use `pgregory.net/rapid` for random PEM generation
    - **Validates: Requirements 3.1, 3.2, 3.5**

  - [ ]* 3.5 Write property test: Standard OpenVPN structural invariant (Property 3)
    - **Property 3: Standard OpenVPN structural invariant**
    - Verify standard config contains auth-user-pass and does NOT contain <cert> or <key> blocks
    - **Validates: Requirement 3.5 inverse, 7.2**

  - [ ]* 3.6 Write property test: OpenVPN config round-trip (Property 7)
    - **Property 7: OpenVPN config round-trip**
    - Generate random PEM content; verify rendering then extracting inline blocks yields identical PEM
    - **Validates: Requirement 3.5**

  - [ ] 3.7 Write unit tests for L2TP and IKEv2 renderers
    - Verify L2TP output is valid XML with SharedSecret auth
    - Verify IKEv2 output references certificate UUID
    - Verify L2TP returns error when PSK empty
    - _Requirements: 4.1, 4.3, 5.1, 5.3_

- [ ] 4. Checkpoint — Ensure all core package tests pass
  - Run `go test ./panel/internal/passwordless/...` and verify all pass

- [ ] 5. API integration — Portal profile download with passwordless support
  - [ ] 5.1 Modify `portalProfiles` handler to add `passwordless_available` field
    - For each profile (openvpn, l2tp, ikev2): check isPasswordlessEnabled AND planAllowsPasswordless
    - Add `"passwordless_available": true/false` to each profile object
    - Do NOT add for WireGuard profiles
    - _Requirements: 7.1, 7.4, 6.1, 6.2_

  - [ ] 5.2 Modify `portalProfileDownload` handler to support passwordless mode
    - Check for `?passwordless=1` query parameter
    - If present: validate global toggle (403 passwordless_disabled), validate plan (403 plan_passwordless_not_allowed)
    - Call CertManager.GetOrCreateCert for OpenVPN/IKEv2
    - Route to appropriate passwordless renderer
    - _Requirements: 11.1, 11.2, 11.3, 11.4_

  - [ ] 5.3 Implement passwordless OpenVPN download path
    - Read CA cert, tls-crypt key from filesystem (same paths as standard config)
    - Call RenderPasswordlessOVPN with customer cert/key
    - Serve with same Content-Type and filename pattern
    - _Requirements: 3.1, 3.2, 3.3, 3.4_

  - [ ] 5.4 Implement passwordless L2TP download path
    - Read IPSec PSK from vpn_core_settings (existing query)
    - Call RenderPasswordlessL2TP
    - Return 400 if PSK not configured
    - _Requirements: 4.1, 4.2, 4.3_

  - [ ] 5.5 Implement passwordless IKEv2 download path
    - Call CertManager.GetOrCreateCert for ikev2 protocol
    - Create PKCS#12 bundle, call RenderPasswordlessIKEv2
    - _Requirements: 5.1, 5.2, 5.4_

  - [ ]* 5.6 Write property test: Permission gate enforcement (Property 4)
    - **Property 4: Permission gate enforcement**
    - Generate random combinations of global toggle (0/1) and plan flag (0/1); verify 403 when either is 0
    - Use sqlmock to simulate DB responses
    - **Validates: Requirements 1.5, 2.5, 11.3, 11.4**

  - [ ]* 5.7 Write property test: Standard config unchanged (Property 1)
    - **Property 1: Standard config unchanged when passwordless not requested**
    - Verify non-passwordless requests produce identical output to current implementation
    - **Validates: Requirements 7.2, 11.2**

- [ ] 6. Certificate revocation API and CRL distribution
  - [ ] 6.1 Add revoke-cert route registration in api.go
    - `mux.HandleFunc("/api/customers/", ...)` — extend customerByID to handle `/revoke-cert` action suffix
    - Or add dedicated route pattern matching
    - _Requirements: 10.1_

  - [ ] 6.2 Implement revokeCert handler
    - Extract customer ID from path
    - Query active client_certificate for customer
    - Call CertManager.RevokeCert
    - Call CertManager.RegenerateCRL
    - Dispatch "crl.update" node task to all OpenVPN-enabled nodes with CRL PEM in payload
    - Add audit log entry
    - _Requirements: 10.2, 10.3_

  - [ ] 6.3 Add OpenVPN CRL directive to server template defaults
    - Add `crl-verify /etc/openvpn/server/crl.pem` to the openvpn default template in `panel/internal/templates/defaults.go`
    - _Requirements: 10.3_

  - [ ] 6.4 Implement "crl.update" task handler in node agent
    - Accept payload with `crl_pem` field
    - Write content to `/etc/openvpn/server/crl.pem`
    - Send SIGUSR1 to OpenVPN process for CRL reload (or systemctl reload)
    - _Requirements: 10.3_

  - [ ] 6.5 Write test for revocation endpoint with sqlmock
    - Test successful revocation flow
    - Test 404 when no active cert exists
    - Test audit log is created
    - _Requirements: 10.1, 10.2, 10.3_

- [ ] 7. Certificate expiry background worker
  - [ ] 7.1 Add periodic goroutine in `panel/cmd/panel/main.go`
    - Run CertManager.CleanupExpired() every hour (similar to existing cert rotation worker pattern)
    - Log count of expired certificates on each run
    - _Requirements: 9.5_

- [ ] 8. Checkpoint — Ensure all backend tests pass
  - Run `go test ./...` and verify all pass

- [ ] 9. Admin dashboard — Settings and plan UI
  - [ ] 9.1 Add "Passwordless Configs" section to Settings → General view
    - Toggle: "Allow Passwordless Configs" bound to allow_passwordless_configs setting
    - Number input: "Config Expiry (days)" bound to passwordless_config_expiry_days, visible only when toggle enabled
    - Wire save to PATCH /api/panel-settings
    - _Requirements: 1.1, 8.1_

  - [ ] 9.2 Add "Allow Passwordless Configs" checkbox to plan create/edit form
    - Checkbox in plan form bound to allow_passwordless field
    - Disabled with tooltip when global toggle is off (check via /api/public-settings or panel-settings)
    - _Requirements: 2.1, 2.5_

  - [ ] 9.3 Add "Revoke Passwordless Certificate" button to customer detail page
    - Show button when customer has active cert (check via customer detail response or lightweight API call)
    - Confirmation dialog before calling POST /api/customers/{id}/revoke-cert
    - Success/error toast feedback
    - _Requirements: 10.1_

- [ ] 10. Customer portal — Config type selector
  - [ ] 10.1 Create `panel/web/portal/src/components/ConfigTypeSelector.vue`
    - Segmented toggle or radio: "Standard" / "Passwordless"
    - Default to "Standard"
    - Emit selected type to parent
    - Info tooltip: "Passwordless configs use certificates. Easier for device setup."
    - _Requirements: 7.1, 7.2, 7.5_

  - [ ] 10.2 Integrate ConfigTypeSelector into Profiles view
    - Render selector per profile when `passwordless_available` is true
    - Hide for WireGuard profiles
    - When "Passwordless" selected, append `?passwordless=1` to download URL
    - _Requirements: 7.1, 7.3, 7.4, 6.1, 6.2_

- [ ] 11. Final integration testing
  - [ ] 11.1 Write integration test: enable globally + enable on plan + download passwordless OpenVPN → verify no auth-user-pass, valid cert blocks
    - _Requirements: 3.1, 3.2, 3.5, 7.1_

  - [ ] 11.2 Write integration test: global toggle disabled → 403 on passwordless request
    - _Requirements: 1.5, 11.3_

  - [ ] 11.3 Write integration test: plan disallows → 403 on passwordless request
    - _Requirements: 2.5, 11.4_

  - [ ] 11.4 Write integration test: revoke cert → new download generates fresh cert with new serial
    - _Requirements: 10.2, 10.4_

  - [ ] 11.5 Write regression test: standard config download unchanged (verify auth-user-pass present, no <cert>/<key>)
    - _Requirements: 7.2_

- [ ] 12. Final checkpoint — All tests pass
  - Run `go test ./...`, `cd panel/web/admin && npm run test`, `cd panel/web/portal && npm run test`

## Notes

- Tasks marked with `*` are property-based tests (optional for faster MVP, recommended for correctness)
- Property tests use `pgregory.net/rapid` for Go and `fast-check` for TypeScript as specified in the tech stack
- The PANEL_CERT_ENCRYPTION_KEY env var must be documented in install.sh and deploy.sh
- WireGuard is excluded from this feature since it is inherently passwordless (key-based)
- The existing `panelSettings` handler requires no changes — it already supports arbitrary key-value pairs
- CRL verification requires OpenVPN server restart/reload; the node agent handles this via task system
- PKCS#12 generation for IKEv2 uses Go's `crypto/pkcs12` (available in `software.sslmate.com/src/go-pkcs12`)

## Task Dependency Graph

```json
{
  "waves": [
    { "id": 0, "tasks": ["1.1"] },
    { "id": 1, "tasks": ["1.2", "1.3", "2.1"] },
    { "id": 2, "tasks": ["2.2", "2.3", "2.4", "2.5", "2.6", "2.7", "2.8"] },
    { "id": 3, "tasks": ["3.1", "3.2", "3.3"] },
    { "id": 4, "tasks": ["3.4", "3.5", "3.6", "3.7"] },
    { "id": 5, "tasks": ["5.1", "5.2", "5.3", "5.4", "5.5"] },
    { "id": 6, "tasks": ["5.6", "5.7", "6.1", "6.2", "6.3", "6.4", "6.5"] },
    { "id": 7, "tasks": ["7.1"] },
    { "id": 8, "tasks": ["9.1", "9.2", "9.3"] },
    { "id": 9, "tasks": ["10.1", "10.2"] },
    { "id": 10, "tasks": ["11.1", "11.2", "11.3", "11.4", "11.5"] }
  ]
}
```

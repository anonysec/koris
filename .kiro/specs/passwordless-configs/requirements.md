# Requirements Document

## Introduction

Add optional passwordless VPN configuration generation to KorisPanel. Currently all VPN profiles require username/password authentication (auth-user-pass for OpenVPN, user credentials for L2TP/IKEv2). This feature introduces an alternative mode where configurations use certificate-based or pre-shared key authentication without embedding user credentials, making device setup easier for non-technical users. The feature is controlled by a global admin toggle and per-plan permission flags, ensuring administrators retain full control over which customers can access passwordless configs.

## Glossary

- **Panel**: The central KorisPanel web application managing nodes, customers, and VPN services
- **Admin_UI**: The Vue 3 admin dashboard SPA for system management
- **Customer_Portal**: The Vue 3 customer-facing SPA for self-service VPN profile downloads
- **Config_Generator**: The backend component responsible for rendering VPN configuration files for download
- **Passwordless_Config**: A VPN configuration file that authenticates using certificates or pre-shared keys instead of username/password credentials
- **Standard_Config**: A VPN configuration file that requires username/password authentication (current default behavior)
- **Panel_Settings**: The panel_settings database table storing global key-value configuration pairs
- **Plan**: A subscription tier that defines data limits, speed, duration, and feature access for customers
- **Client_Certificate**: An X.509 certificate issued to a specific customer for OpenVPN certificate-based authentication
- **PSK**: A pre-shared key used for L2TP/IPSec authentication without user credentials
- **Config_Expiry**: The validity period after which a passwordless configuration must be re-downloaded or becomes invalid

## Requirements

### Requirement 1: Global Passwordless Toggle

**User Story:** As an administrator, I want to enable or disable the passwordless config option globally, so that I can control whether the feature is available across the entire panel.

#### Acceptance Criteria

1. THE Admin_UI SHALL display a "Allow Passwordless Configs" toggle in the Settings → General section
2. WHEN an administrator enables the passwordless toggle, THE Panel SHALL store the setting with key "allow_passwordless_configs" and value "1" in the Panel_Settings table
3. WHEN an administrator disables the passwordless toggle, THE Panel SHALL store the setting with key "allow_passwordless_configs" and value "0" in the Panel_Settings table
4. THE Panel SHALL default to passwordless configs disabled (value "0") when the setting does not exist in the Panel_Settings table
5. WHEN the global passwordless toggle is disabled, THE Customer_Portal SHALL hide the passwordless download option for all customers regardless of plan settings

### Requirement 2: Per-Plan Passwordless Permission

**User Story:** As an administrator, I want to allow passwordless configs for specific plans only, so that I can offer the feature as a premium option or restrict it to trusted plans.

#### Acceptance Criteria

1. THE Admin_UI SHALL display an "Allow Passwordless Configs" checkbox on the plan create and edit forms
2. WHEN an administrator creates or updates a plan with the passwordless checkbox enabled, THE Panel SHALL store the allow_passwordless flag as 1 in the plans table for that plan
3. WHEN an administrator creates or updates a plan with the passwordless checkbox disabled, THE Panel SHALL store the allow_passwordless flag as 0 in the plans table for that plan
4. THE Panel SHALL default the allow_passwordless flag to 0 for new plans
5. WHILE the global passwordless toggle is disabled, THE Admin_UI SHALL display the per-plan checkbox as disabled with a tooltip indicating the global setting must be enabled first

### Requirement 3: Passwordless OpenVPN Config Generation

**User Story:** As a customer, I want to download an OpenVPN config that does not require me to enter a username and password, so that I can use certificate-based authentication for easier device setup.

#### Acceptance Criteria

1. WHEN a customer requests a passwordless OpenVPN config, THE Config_Generator SHALL produce a valid .ovpn file without the "auth-user-pass" directive
2. WHEN a customer requests a passwordless OpenVPN config, THE Config_Generator SHALL embed the Client_Certificate and client private key inline using <cert> and <key> blocks
3. THE Config_Generator SHALL include the CA certificate, tls-crypt key, and all standard cipher/auth directives in passwordless OpenVPN configs identical to Standard_Configs
4. IF a Client_Certificate does not exist for the requesting customer, THEN THE Panel SHALL generate a new Client_Certificate signed by the panel CA before producing the config
5. FOR ALL valid passwordless OpenVPN configs, the generated file SHALL contain <cert>, <key>, and <ca> blocks and SHALL NOT contain the "auth-user-pass" directive (structural property)

### Requirement 4: Passwordless L2TP Config Generation

**User Story:** As a customer, I want to download an L2TP/IPSec config that authenticates using only a pre-shared key, so that I can set up the VPN on devices without entering separate credentials.

#### Acceptance Criteria

1. WHEN a customer requests a passwordless L2TP config, THE Config_Generator SHALL produce a .mobileconfig file with IPSec PSK authentication and no EAP/username fields
2. THE Config_Generator SHALL embed the system-wide IPSec PSK in the passwordless L2TP configuration
3. IF the system IPSec PSK is not configured, THEN THE Panel SHALL return an error indicating L2TP passwordless configs are unavailable
4. THE Config_Generator SHALL set the AuthenticationMethod to "SharedSecret" in the passwordless L2TP .mobileconfig payload

### Requirement 5: Passwordless IKEv2 Config Generation

**User Story:** As a customer, I want to download an IKEv2 config that uses certificate authentication, so that I can connect without entering credentials on Apple/Windows devices.

#### Acceptance Criteria

1. WHEN a customer requests a passwordless IKEv2 config, THE Config_Generator SHALL produce a .mobileconfig file with certificate-based authentication instead of EAP-MSCHAPv2
2. THE Config_Generator SHALL embed the Client_Certificate in PKCS#12 format within the IKEv2 .mobileconfig payload
3. THE Config_Generator SHALL set AuthenticationMethod to "Certificate" in the IKEv2 VPN payload
4. IF a Client_Certificate does not exist for the requesting customer, THEN THE Panel SHALL generate a new Client_Certificate signed by the panel CA before producing the config

### Requirement 6: WireGuard Config Handling

**User Story:** As a customer, I want WireGuard configs to remain unchanged since they are already passwordless, so that there is no confusion about config types for WireGuard.

#### Acceptance Criteria

1. THE Customer_Portal SHALL NOT display a passwordless toggle for WireGuard profiles since WireGuard is inherently key-based
2. THE Panel SHALL treat all WireGuard configs as passwordless by nature without requiring the passwordless feature flag

### Requirement 7: Customer Portal Config Selection

**User Story:** As a customer, I want to choose between standard (username/password) and passwordless configs when downloading, so that I can pick the authentication method that suits my device.

#### Acceptance Criteria

1. WHILE the global passwordless toggle is enabled AND the customer's plan allows passwordless configs, THE Customer_Portal SHALL display a toggle or selector allowing the customer to choose between "Standard" and "Passwordless" config types
2. THE Customer_Portal SHALL default the config type selector to "Standard" to preserve existing behavior
3. WHEN a customer selects the passwordless option, THE Customer_Portal SHALL request the config download with a "passwordless=1" query parameter
4. IF the customer's plan does not allow passwordless configs, THEN THE Customer_Portal SHALL hide the passwordless option and display only standard config downloads
5. THE Customer_Portal SHALL display a brief explanation text indicating that passwordless configs use certificates instead of username/password

### Requirement 8: Config Expiry for Passwordless Configs

**User Story:** As an administrator, I want passwordless configs to have a configurable expiry period, so that revoked or expired certificates do not remain usable indefinitely.

#### Acceptance Criteria

1. THE Admin_UI SHALL provide a "Passwordless Config Expiry (days)" input field in the Settings → General section alongside the global toggle
2. THE Panel SHALL default the config expiry to 30 days when no value is configured
3. WHEN a Client_Certificate is generated, THE Panel SHALL set the certificate NotAfter date to the current time plus the configured expiry days
4. THE Panel SHALL store the expiry setting with key "passwordless_config_expiry_days" in the Panel_Settings table
5. IF an administrator sets the expiry to 0, THEN THE Panel SHALL generate certificates with no expiry (matching the customer's subscription end date instead)

### Requirement 9: Certificate Generation and Storage

**User Story:** As the system, I want to generate and store client certificates for passwordless authentication, so that each customer has a unique cryptographic identity.

#### Acceptance Criteria

1. WHEN the Panel generates a Client_Certificate, THE Panel SHALL create an RSA 2048-bit key pair and X.509 certificate signed by the existing panel CA
2. THE Panel SHALL store the encrypted client private key, certificate PEM, serial number, issued_at, expires_at, and customer_id in a client_certificates table
3. THE Panel SHALL ensure each customer has at most one active Client_Certificate per protocol at any time
4. WHEN a customer's subscription expires or is terminated, THE Panel SHALL revoke the associated Client_Certificate by updating its status to "revoked"
5. IF a customer requests a passwordless config and their existing certificate has expired, THEN THE Panel SHALL generate a new Client_Certificate automatically

### Requirement 10: Certificate Revocation

**User Story:** As an administrator, I want to revoke a customer's passwordless certificates, so that I can immediately block access if needed.

#### Acceptance Criteria

1. THE Admin_UI SHALL display a "Revoke Passwordless Cert" action on the customer detail page when a customer has an active Client_Certificate
2. WHEN an administrator revokes a certificate, THE Panel SHALL update the certificate status to "revoked" and set revoked_at timestamp
3. WHEN a certificate is revoked, THE Panel SHALL regenerate the OpenVPN CRL (Certificate Revocation List) file and trigger a reload on affected nodes
4. IF a customer requests a new passwordless config after revocation, THEN THE Panel SHALL generate a fresh Client_Certificate

### Requirement 11: API Endpoint for Passwordless Config Download

**User Story:** As the system, I want a clear API contract for passwordless config downloads, so that the frontend can reliably request the correct config type.

#### Acceptance Criteria

1. WHEN the portal profile download endpoint receives a request with query parameter "passwordless=1", THE Panel SHALL return the passwordless variant of the requested config
2. WHEN the portal profile download endpoint receives a request without the "passwordless" parameter, THE Panel SHALL return the standard config with username/password authentication
3. IF a customer requests a passwordless config but the feature is disabled globally, THEN THE Panel SHALL return a 403 response with error "passwordless_disabled"
4. IF a customer requests a passwordless config but their plan does not allow it, THEN THE Panel SHALL return a 403 response with error "plan_passwordless_not_allowed"

### Requirement 12: Database Schema Changes

**User Story:** As a developer, I want the database schema to support passwordless config data, so that certificates and settings are properly persisted.

#### Acceptance Criteria

1. THE Panel SHALL add an "allow_passwordless" TINYINT(1) DEFAULT 0 column to the plans table via a numbered SQL migration
2. THE Panel SHALL create a client_certificates table with columns: id, customer_id, protocol, private_key_encrypted, certificate_pem, serial_number, status (active/revoked/expired), issued_at, expires_at, revoked_at, created_at
3. THE Panel SHALL create an index on client_certificates(customer_id, protocol, status) for efficient lookups
4. THE Panel SHALL add the "allow_passwordless_configs" and "passwordless_config_expiry_days" entries to Panel_Settings via the migration with default values "0" and "30"


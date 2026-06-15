# KorisPanel UI/UX Direction

This frontend uses a compact premium SaaS/dashboard style optimized for VPN operations.

## Product mood

- Professional network operations, not consumer entertainment.
- Dense but readable admin workflows.
- Dark command-center surface with cyan/blue signal accents.
- Clear operational status language: online, active, limited, expired, disabled.

## Design rules

- Keep primary actions obvious and close to the user's current task.
- Avoid decorative icons/emojis for critical controls; use text, status pills, and simple glyphs.
- Tables must stay compact, scannable, and responsive.
- Every form control needs visible focus state and clear error state.
- Empty states should explain the next action.
- Respect reduced motion.

## Tokens

- Background: `#070a12`, `#0b1120`
- Surface: rgba slate panels with subtle glass depth
- Border: low-contrast slate line, stronger on hover/focus
- Primary: `#2563eb`
- Accent: `#22d3ee`
- Success: `#22c55e`
- Warning: `#f59e0b`
- Danger: `#ef4444`

## Core components in use

- Auth hero + auth card
- App shell/sidebar/topbar
- Score/readiness chip
- Metric cards
- Operational flow / journey steps
- Data table with status summary
- Status pills
- Form panels
- Empty and alert states

## Next UI components to add

- Customer detail drawer/page
- Plan cards + plan CRUD table
- Payment review queue
- Ticket conversation panel
- Node telemetry cards
- Global command/search palette

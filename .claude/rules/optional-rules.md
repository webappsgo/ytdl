# Optional Features Rules (PART 34-36)

⚠️ **These rules are NON-NEGOTIABLE. Violations are bugs.** ⚠️

**These features are OPTIONAL, but when implemented they are NON-NEGOTIABLE.**

## CRITICAL - NEVER DO

- NEVER implement partial multi-user (all or nothing)
- NEVER implement partial organizations (all or nothing)
- NEVER implement partial custom domains (all or nothing)
- NEVER gate features behind paid tiers

## CRITICAL - ALWAYS DO (when implementing)

- ALWAYS implement full spec when enabling any optional feature
- ALWAYS support user registration modes: public, private, disabled
- ALWAYS implement role-based access control with multi-user
- ALWAYS support organization hierarchies with orgs feature
- ALWAYS support custom domain verification and SSL

## Optional Features

| Feature | PART | Status |
|---------|------|--------|
| Multi-User | 34 | OPTIONAL - full RBAC when enabled |
| Organizations | 35 | OPTIONAL - requires multi-user |
| Custom Domains | 36 | OPTIONAL - requires organizations |

## Activation

To activate: change AI.md PART status from OPTIONAL to REQUIRED.

For complete details, see AI.md PART 34, 35, 36

# API Rules (PART 13, 14, 15)

⚠️ **These rules are NON-NEGOTIABLE. Violations are bugs.** ⚠️

## CRITICAL - NEVER DO

- NEVER expose sensitive info in health endpoints
- NEVER return stack traces in production API responses
- NEVER use bare paths in embedded code - use FQDN
- NEVER skip CORS configuration

## CRITICAL - ALWAYS DO

- ALWAYS have /healthz and /api/v1/healthz endpoints
- ALWAYS use consistent JSON response format
- ALWAYS use proper HTTP status codes
- ALWAYS support content negotiation (HTML vs JSON)
- ALWAYS include Swagger/OpenAPI documentation
- ALWAYS include GraphQL endpoint
- ALWAYS support SSL/TLS with Let's Encrypt

## Endpoint Pattern

| Web Route (HTML) | API Route (JSON) |
|------------------|------------------|
| `/` | `/api/v1/` |
| `/healthz` | `/api/v1/healthz` |
| `/{admin_path}/dashboard` | `/api/v1/{admin_path}/dashboard` |
| `/openapi` | `/openapi.json` |
| `/graphql` | `/api/v1/graphql` |

## Key Rules

| Rule | Description |
|------|-------------|
| **Every page = API** | For every web page, there's a corresponding API endpoint |
| **JSON format** | Consistent structure with error codes |
| **Health check** | Returns status only, no sensitive info |
| **SSL/TLS** | Let's Encrypt auto-renewal support |
| **Swagger** | Always at /openapi (root level) |
| **GraphQL** | Always at /graphql and /api/v1/graphql |

For complete details, see AI.md PART 13, 14, 15

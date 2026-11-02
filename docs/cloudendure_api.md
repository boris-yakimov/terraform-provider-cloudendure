# CloudEndure API notes

Reference: <https://aws.amazon.com/blogs/storage/interacting-with-the-cloudendure-api-using-postman/>

Base URL: `https://console.cloudendure.com/api/latest`

## Authentication

Generate an API token in the CloudEndure console under
*Setup & Info -> Other Settings -> API Token*.

```http
POST /login
Content-Type: application/json

{"userApiToken": "<API_TOKEN>"}
```

Returns `200` (or `204`) and sets two cookies, valid for about one hour:

- `session` - the authenticated session
- `XSRF-TOKEN` - anti-CSRF token, e.g. `"AbCdEf==\012"` (quoted, with a Python-style escape)

Every subsequent request must send both cookies back and the XSRF token in a header:

```http
Cookie: XSRF-TOKEN="<XSRF>"; session=<SESSION>
X-XSRF-TOKEN: <XSRF>
```

Go's `net/http/cookiejar` silently drops the `XSRF-TOKEN` cookie because the
`\012` escape contains a backslash, which is not a valid cookie-value byte.
The client in `internal/client` therefore parses `Set-Cookie` headers itself.

## Endpoints used by the provider

| Method | Path | Description |
| ------ | ---- | ----------- |
| GET    | `/projects/{projectId}/machines` | List machines |
| GET    | `/projects/{projectId}/machines/{machineId}` | Get machine |
| GET    | `/projects/{projectId}/blueprints` | List blueprints |
| POST   | `/projects/{projectId}/blueprints` | Create blueprint |
| GET    | `/projects/{projectId}/blueprints/{blueprintId}` | Get blueprint |
| PATCH  | `/projects/{projectId}/blueprints/{blueprintId}` | Configure blueprint |
| GET    | `/projects/{projectId}/recoveryPlans` | List recovery plans |
| POST   | `/projects/{projectId}/recoveryPlans` | Create recovery plan |
| GET    | `/projects/{projectId}/recoveryPlans/{recoveryPlanId}` | Get recovery plan |
| PATCH  | `/projects/{projectId}/recoveryPlans/{recoveryPlanId}` | Update recovery plan |
| DELETE | `/projects/{projectId}/recoveryPlans/{recoveryPlanId}` | Delete recovery plan |

There is no `DELETE` for blueprints: a blueprint belongs to a machine and lives
as long as the machine does.

### Example: configure blueprint

```http
PATCH /projects/{projectId}/blueprints/{blueprintId}

{
  "id": "<BLUEPRINT_ID>",
  "machineId": "<MACHINE_ID>",
  "subnetIDs": ["subnet-0123456789abcdef0"],
  "securityGroupIDs": ["sg-0123456789abcdef0"]
}
```

Returns `200` with the full, updated blueprint.

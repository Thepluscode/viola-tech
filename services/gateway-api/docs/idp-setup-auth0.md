# Auth0 Setup Guide

Complete guide to configure Viola Gateway API with Auth0.

---

## Prerequisites

- Auth0 account (free tier works for testing)
- Viola Gateway API deployed and accessible

---

## Step 1: Create API

1. **Navigate to Applications → APIs**
   - Go to https://manage.auth0.com
   - Click **Applications** → **APIs**
   - Click **Create API**

2. **Configure API**
   - **Name**: `Viola Gateway API`
   - **Identifier**: `https://api.viola.com` (your API URL or any unique identifier)
   - **Signing Algorithm**: RS256
   - Click **Create**

3. **Note Configuration**
   - **Identifier** (this is your Audience): `https://api.viola.com`
   - **Signing Algorithm**: RS256

---

## Step 2: Add Permissions (Scopes)

1. **Navigate to Permissions Tab**
   - In your API settings, click **Permissions**
   - Add the following permissions:

   | Permission (Scope) | Description |
   |--------------------|-------------|
   | `incidents:read` | View security incidents |
   | `incidents:write` | Update security incidents |
   | `alerts:read` | View security alerts |
   | `alerts:write` | Update security alerts |
   | `rules:read` | View detection rules |
   | `rules:write` | Manage detection rules |
   | `admin:*` | Full administrative access |

2. **For Each Permission**
   - Click **Add**
   - Enter permission (scope) and description
   - Permissions are automatically included in access tokens when requested

---

## Step 3: Create Machine-to-Machine Application

For service-to-service authentication:

1. **Navigate to Applications**
   - Click **Applications** → **Applications**
   - Click **Create Application**

2. **Configure Application**
   - **Name**: `Viola Gateway API Service`
   - **Type**: Machine to Machine Applications
   - Click **Create**

3. **Authorize Application**
   - Select **Viola Gateway API** (created in Step 1)
   - Select all permissions you want to grant
   - Click **Authorize**

4. **Note Credentials**
   - Copy **Client ID**
   - Copy **Client Secret**
   - **Domain**: `your-tenant.auth0.com`

---

## Step 4: Create Single Page/Web Application

For user authentication:

1. **Create Application**
   - Click **Applications** → **Applications** → **Create Application**
   - **Name**: `Viola Gateway Web`
   - **Type**: Single Page Web Applications (or Regular Web Applications)
   - Click **Create**

2. **Configure Application**
   - **Allowed Callback URLs**: `https://your-frontend.com/callback`
   - **Allowed Logout URLs**: `https://your-frontend.com`
   - **Allowed Web Origins**: `https://your-frontend.com`
   - **Allowed Origins (CORS)**: `https://your-frontend.com`
   - Click **Save Changes**

3. **Configure Grant Types**
   - Scroll to **Advanced Settings** → **Grant Types**
   - Enable:
     - Authorization Code
     - Refresh Token
     - Implicit (only if needed)
   - Click **Save Changes**

---

## Step 5: Create Roles (For RBAC)

1. **Navigate to User Management → Roles**
   - Click **User Management** → **Roles**
   - Click **Create Role**

2. **Create Roles**

   | Role Name | Description |
   |-----------|-------------|
   | `SOCReader` | View-only access to incidents and alerts |
   | `SOCResponder` | Triage and respond to incidents |
   | `SOCEngineer` | Create and manage detection rules |
   | `ViolaAdmin` | Full administrative access |

3. **Assign Permissions to Roles**
   - For each role, click the role name
   - Click **Permissions** tab → **Add Permissions**
   - Select **Viola Gateway API**
   - Select appropriate permissions:
     - **SOCReader**: `incidents:read`, `alerts:read`
     - **SOCResponder**: `incidents:read`, `incidents:write`, `alerts:read`, `alerts:write`
     - **SOCEngineer**: All reader/responder permissions + `rules:read`, `rules:write`
     - **ViolaAdmin**: `admin:*` (or all permissions)
   - Click **Add Permissions**

---

## Step 6: Create Users and Assign Roles

1. **Create Users**
   - Go to **User Management** → **Users**
   - Click **Create User**
   - Enter email and password
   - Click **Create**

2. **Assign Roles**
   - Click on the user
   - Go to **Roles** tab
   - Click **Assign Roles**
   - Select appropriate role(s)
   - Click **Assign**

---

## Step 7: Add Custom Claims (Rules/Actions)

### Method 1: Using Actions (Recommended - New)

1. **Navigate to Actions → Flows**
   - Click **Actions** → **Flows**
   - Select **Login**

2. **Create Action**
   - Click **Custom** tab on the right
   - Click **Create Action**
   - **Name**: `Add Viola Claims`
   - **Trigger**: Login / Post Login
   - Click **Create**

3. **Add Code**

```javascript
/**
* Handler that will be called during the execution of a PostLogin flow.
*
* @param {Event} event - Details about the user and the context in which they are logging in.
* @param {PostLoginAPI} api - Interface whose methods can be used to change the behavior of the login.
*/
exports.onExecutePostLogin = async (event, api) => {
  const namespace = 'https://viola.com/';

  // Add roles claim
  if (event.authorization) {
    api.accessToken.setCustomClaim(`${namespace}roles`, event.authorization.roles || []);
  }

  // Add tenant ID (from user metadata)
  if (event.user.app_metadata && event.user.app_metadata.tenantId) {
    api.accessToken.setCustomClaim(`${namespace}tid`, event.user.app_metadata.tenantId);
  }

  // Add email
  if (event.user.email) {
    api.accessToken.setCustomClaim(`${namespace}email`, event.user.email);
  }

  // Add permissions (scopes)
  if (event.authorization && event.authorization.permissions) {
    api.accessToken.setCustomClaim(`${namespace}permissions`, event.authorization.permissions);
  }
};
```

4. **Deploy Action**
   - Click **Deploy**
   - Back in **Flows** → **Login**, drag your action to the flow
   - Click **Apply**

### Method 2: Using Rules (Legacy)

1. **Navigate to Auth Pipeline → Rules**
   - Click **Auth Pipeline** → **Rules**
   - Click **Create Rule** → **Empty Rule**

2. **Add Rule**
   - **Name**: `Add Viola Claims`
   - **Code**:

```javascript
function addViolaClaims(user, context, callback) {
  const namespace = 'https://viola.com/';

  // Add roles
  context.accessToken[namespace + 'roles'] = (context.authorization && context.authorization.roles) || [];

  // Add tenant ID from user metadata
  if (user.app_metadata && user.app_metadata.tenantId) {
    context.accessToken[namespace + 'tid'] = user.app_metadata.tenantId;
  }

  // Add email
  if (user.email) {
    context.accessToken[namespace + 'email'] = user.email;
  }

  // Add permissions
  if (context.authorization && context.authorization.permissions) {
    context.accessToken[namespace + 'permissions'] = context.authorization.permissions;
  }

  callback(null, user, context);
}
```

3. **Save Rule**

---

## Step 8: Add User Metadata (Tenant ID)

1. **Edit User**
   - Go to **User Management** → **Users**
   - Click on a user

2. **Add App Metadata**
   - Scroll to **app_metadata** section
   - Click **Edit**
   - Add:

```json
{
  "tenantId": "tenant-abc123"
}
```

   - Click **Save**

---

## Step 9: Enable MFA

1. **Navigate to Security → Multi-factor Auth**
   - Click **Security** → **Multi-factor Auth**
   - Enable desired factors:
     - One-time Password (recommended)
     - SMS
     - Push Notifications (Guardian)
     - Email
     - WebAuthn with FIDO2 Security Keys

2. **Configure Policy**
   - Under **Policies**, select:
     - **Always**: Require MFA for all users
     - **Adaptive**: Use risk-based MFA
     - **Never**: No MFA (not recommended)

3. **Result**
   - When users authenticate with MFA, the `amr` claim will include the method
   - Example: `["pwd", "otp"]` for password + one-time password

---

## Step 10: Get Configuration Values

### Issuer URL
```
https://<YOUR_DOMAIN>.auth0.com/
```

**Example:**
```
https://viola-dev.auth0.com/
```

### Audience
```
https://api.viola.com
```

(Whatever you set as API Identifier in Step 1)

### JWKS URL (Auto-discovered)
Gateway will discover from:
```
https://<YOUR_DOMAIN>.auth0.com/.well-known/openid-configuration
```

**Discovered JWKS URL:**
```
https://<YOUR_DOMAIN>.auth0.com/.well-known/jwks.json
```

---

## Step 11: Configure Gateway API

```bash
# OIDC Configuration
OIDC_ISSUER_URL="https://viola-dev.auth0.com/"
OIDC_AUDIENCE="https://api.viola.com"
AUTH_REQUIRE_BEARER="true"
AUTH_ALLOW_ALGOS="RS256"
OIDC_CLOCK_SKEW_SECONDS="120"

# Rate Limiting
RATE_LIMIT_ENABLED="true"
RATE_LIMIT_PER_MIN_DEFAULT="120"
RATE_LIMIT_KEY_CLAIMS="sub,https://viola.com/tid"

# Audit
AUDIT_KAFKA_BROKER="localhost:9092"
AUDIT_TOPIC="viola.prod.audit.v1.event"

# Database
PG_HOST="localhost"
PG_PORT="5432"
PG_USER="postgres"
PG_PASSWORD="<your-password>"
PG_DATABASE="viola_gateway"

# Server
PORT="8080"
VIOLA_ENV="prod"
```

**Note:** For custom claims with namespaces, use the full claim name in `RATE_LIMIT_KEY_CLAIMS`.

---

## Step 12: Test Authentication

### Get Access Token (Client Credentials Flow)

```bash
# Service-to-service authentication
curl -X POST \
  https://viola-dev.auth0.com/oauth/token \
  -H "Content-Type: application/json" \
  -d '{
    "grant_type": "client_credentials",
    "client_id": "<YOUR_CLIENT_ID>",
    "client_secret": "<YOUR_CLIENT_SECRET>",
    "audience": "https://api.viola.com"
  }'
```

### Get Access Token (Authorization Code Flow)

```bash
# Step 1: Redirect user to authorization URL (in browser)
https://viola-dev.auth0.com/authorize?\
  response_type=code&\
  client_id=<YOUR_CLIENT_ID>&\
  redirect_uri=https://your-app.com/callback&\
  scope=openid%20incidents:read%20alerts:read&\
  audience=https://api.viola.com&\
  state=random_state_string

# Step 2: Exchange code for token
curl -X POST \
  https://viola-dev.auth0.com/oauth/token \
  -H "Content-Type: application/json" \
  -d '{
    "grant_type": "authorization_code",
    "client_id": "<YOUR_CLIENT_ID>",
    "client_secret": "<YOUR_CLIENT_SECRET>",
    "code": "<AUTHORIZATION_CODE>",
    "redirect_uri": "https://your-app.com/callback"
  }'
```

### Test the API

```bash
# Extract token from response
export TOKEN="<access_token>"

# List incidents
curl -H "Authorization: Bearer $TOKEN" \
  https://your-gateway.com/api/v1/incidents

# Update incident
curl -X PATCH \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"status": "closed"}' \
  https://your-gateway.com/api/v1/incidents/INC-123
```

### Decode Token (Debugging)

```bash
# Use jwt.io or jwt-cli
echo $TOKEN | jwt decode -

# Or use jq
echo $TOKEN | cut -d. -f2 | base64 -d | jq
```

---

## Troubleshooting

### Error: "Invalid audience"

**Solution:**
- Ensure token request includes `"audience": "https://api.viola.com"`
- Verify `OIDC_AUDIENCE` matches the API identifier exactly

### Error: "Insufficient scope"

**Solution:**
- Check that requested scopes are defined in the API (Step 2)
- For M2M apps, ensure permissions are authorized (Step 3)
- For user apps, ensure scopes are requested in authorization URL

### JWT Missing Custom Claims

**Solution:**
- Verify Action/Rule is enabled and deployed
- Check that namespace is used (`https://viola.com/`)
- Auth0 requires custom claims to have a namespace (URL format)

### JWT Missing Roles

**Solution:**
- Ensure user is assigned a role
- Check Action/Rule includes roles claim
- Verify `event.authorization.roles` is populated

### MFA Not Required

**Solution:**
- Check MFA policy is set to "Always" or "Adaptive"
- Verify at least one MFA factor is enabled
- Check user has enrolled in MFA

---

## Token Claim Mapping

| Auth0 Claim | Viola Field | Description |
|-------------|-------------|-------------|
| `sub` | Subject | User's unique identifier |
| `https://viola.com/email` | Email | User's email (custom claim) |
| `https://viola.com/roles` | Roles | Roles assigned to user (custom claim) |
| `https://viola.com/permissions` | Scopes | Permissions granted (custom claim) |
| `https://viola.com/tid` | TenantID | Tenant identifier (custom claim) |
| `iss` | Issuer | Token issuer URL |
| `aud` | Audience | Intended audience |
| `exp` | Expiry | Token expiration time |
| `amr` | - | Authentication methods (for MFA) |

**Note:** Auth0 requires custom claims to use a namespace (e.g., `https://viola.com/`). Update your claim mapping logic accordingly.

---

## Security Best Practices

1. **Use Namespaced Custom Claims**
   - Auth0 requires custom claims to have a URL namespace
   - Prevents claim collisions

2. **Enable Anomaly Detection**
   - Auth0 → Security → Attack Protection
   - Brute-force protection
   - Breached password detection

3. **Rotate Client Secrets**
   - Use Auth0 Management API to rotate secrets
   - Support multiple active secrets during rotation

4. **Use Refresh Token Rotation**
   - Applications → Settings → Advanced Settings
   - Enable "Rotation" under Refresh Token Behavior

5. **Monitor Logs**
   - Auth0 → Monitoring → Logs
   - Set up log streams to SIEM
   - Configure alerting for suspicious activity

6. **Implement Bot Detection**
   - Use Auth0's Bot Detection feature
   - Protect login endpoints from automated attacks

---

## References

- [Auth0 Documentation](https://auth0.com/docs)
- [Secure API with Auth0](https://auth0.com/docs/get-started/architecture-scenarios/spa-api)
- [Custom Claims](https://auth0.com/docs/secure/tokens/json-web-tokens/create-custom-claims)
- [Roles-Based Access Control](https://auth0.com/docs/manage-users/access-control/rbac)
- [Actions](https://auth0.com/docs/customize/actions)

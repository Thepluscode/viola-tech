# Okta Setup Guide

Complete guide to configure Viola Gateway API with Okta.

---

## Prerequisites

- Okta account with admin access
- Viola Gateway API deployed and accessible

---

## Step 1: Create Authorization Server (Recommended)

For production use, create a dedicated authorization server:

1. **Navigate to Security → API**
   - Go to https://`<your-domain>`.okta.com/admin/access/api
   - Click **Add Authorization Server**

2. **Configure Server**
   - **Name**: `Viola Gateway API`
   - **Audience**: `api://viola-gateway`
   - **Description**: Authorization server for Viola security platform
   - Click **Save**

3. **Note Your Issuer**
   - Example: `https://dev-12345.okta.com/oauth2/aus1234abcd`
   - Or use `default`: `https://dev-12345.okta.com/oauth2/default`

**Alternative:** Use the default authorization server at `/oauth2/default`

---

## Step 2: Add Scopes

1. **Navigate to Scopes Tab**
   - In your authorization server, click **Scopes**
   - Click **Add Scope**

2. **Create Scopes**

   Add the following scopes:

   | Scope Name | Display Name | Description | Default | Metadata |
   |------------|--------------|-------------|---------|----------|
   | `incidents:read` | Read Incidents | View security incidents | No | Public |
   | `incidents:write` | Write Incidents | Update security incidents | No | Public |
   | `alerts:read` | Read Alerts | View security alerts | No | Public |
   | `alerts:write` | Write Alerts | Update security alerts | No | Public |
   | `rules:read` | Read Rules | View detection rules | No | Public |
   | `rules:write` | Write Rules | Manage detection rules | No | Public |

   For each scope:
   - Click **Add Scope**
   - Enter name, display name, and description
   - **User consent**: Required (or leave as default)
   - **Metadata**: Publish to use in tokens
   - Click **Create**

---

## Step 3: Create API Service Application

1. **Navigate to Applications**
   - Go to **Applications** → **Applications**
   - Click **Create App Integration**

2. **Configure Integration**
   - **Sign-in method**: API Services
   - Click **Next**

3. **Configure Application**
   - **App integration name**: `Viola Gateway API Service`
   - Click **Save**

4. **Note Credentials**
   - Copy **Client ID**
   - Copy **Client Secret** (click to reveal)

---

## Step 4: Create Web Application (For User Access)

1. **Create Another App Integration**
   - Go to **Applications** → **Applications**
   - Click **Create App Integration**

2. **Configure Integration**
   - **Sign-in method**: OIDC - OpenID Connect
   - **Application type**: Web Application
   - Click **Next**

3. **Configure Application**
   - **App integration name**: `Viola Gateway Web`
   - **Grant type**: Authorization Code, Refresh Token
   - **Sign-in redirect URIs**: `https://your-frontend.com/callback`
   - **Sign-out redirect URIs**: `https://your-frontend.com/logout`
   - **Controlled access**: Allow everyone in your organization to access
   - Click **Save**

---

## Step 5: Create Groups (For RBAC)

1. **Navigate to Groups**
   - Go to **Directory** → **Groups**
   - Click **Add Group**

2. **Create Security Groups**

   | Group Name | Description |
   |------------|-------------|
   | `Viola-SOC-Readers` | View-only access to incidents and alerts |
   | `Viola-SOC-Responders` | Triage and respond to incidents |
   | `Viola-SOC-Engineers` | Create and manage detection rules |
   | `Viola-Admins` | Full administrative access |

3. **Assign Users**
   - For each group, click the group name
   - Click **Assign people**
   - Search and select users
   - Click **Save**

---

## Step 6: Add Custom Claims

### Add Group Claim

1. **Navigate to Authorization Server → Claims**
   - Click **Add Claim**

2. **Configure Claim**
   - **Name**: `groups`
   - **Include in token type**: Access Token
   - **Value type**: Groups
   - **Filter**: Matches regex: `Viola-.*`
   - **Include in**: Any scope
   - Click **Create**

### Add Tenant ID Claim (For Multi-Tenant)

1. **Add User Attribute**
   - Go to **Directory** → **Profile Editor**
   - Select **User (default)**
   - Click **Add Attribute**
   - **Data type**: string
   - **Display name**: Tenant ID
   - **Variable name**: `tenantId`
   - **Description**: Viola tenant identifier
   - Click **Save**

2. **Add Claim for Tenant ID**
   - Back in Authorization Server → Claims
   - Click **Add Claim**
   - **Name**: `tid`
   - **Include in token type**: Access Token, ID Token
   - **Value**: `user.tenantId`
   - **Include in**: Any scope
   - Click **Create**

### Add Email Claim

1. **Add Email Claim**
   - Authorization Server → Claims → Add Claim
   - **Name**: `email`
   - **Include in token type**: Access Token
   - **Value**: `user.email`
   - **Include in**: Any scope
   - Click **Create**

---

## Step 7: Create Access Policy

1. **Navigate to Access Policies**
   - In your authorization server, click **Access Policies**
   - Click **Add New Access Policy**

2. **Configure Policy**
   - **Name**: `Viola API Access`
   - **Description**: Controls access to Viola Gateway API
   - **Assign to**: Select your Viola applications
   - Click **Create Policy**

3. **Add Rule**
   - Click **Add Rule**
   - **Rule Name**: `Default Access`
   - **Grant type**: All grant types (or select specific ones)
   - **User is**: In groups: `Viola-.*` (regex)
   - **Scopes**: Any scopes (or select specific scopes)
   - **Access token lifetime**: 1 hour
   - **Refresh token lifetime**: 7 days
   - Click **Create Rule**

---

## Step 8: Enable MFA

1. **Navigate to Security → Multifactor**
   - Go to **Security** → **Multifactor**
   - Select factors to enable:
     - Okta Verify (recommended)
     - Google Authenticator
     - SMS Authentication
     - Email Authentication

2. **Create Sign-On Policy**
   - Go to **Security** → **Authentication Policies**
   - Click **Add a Policy**
   - **Name**: `Viola API MFA Required`
   - **Assigned to applications**: Select Viola applications
   - Click **Create Policy and Add Rule**

3. **Configure Rule**
   - **Rule Name**: `Require MFA`
   - **IF User's group membership includes**: `Viola-.*`
   - **AND User's IP is**: Anywhere (or restrict)
   - **THEN Access is**: Allowed after successful authentication
   - **Multifactor authentication**: Required
   - **Re-authentication frequency**: Every sign-in
   - Click **Create Rule**

4. **Result**
   - When users authenticate with MFA, the `amr` claim will include the authentication method (e.g., `["pwd", "otp"]`)

---

## Step 9: Get Configuration Values

### Issuer URL
```
https://<your-domain>.okta.com/oauth2/<authorization-server-id>
```

**For default authorization server:**
```
https://dev-12345.okta.com/oauth2/default
```

### Audience
```
api://viola-gateway
```

(Whatever you set in Step 1)

### JWKS URL (Auto-discovered)
Gateway will discover from:
```
https://dev-12345.okta.com/oauth2/default/.well-known/openid-configuration
```

**Discovered JWKS URL:**
```
https://dev-12345.okta.com/oauth2/default/v1/keys
```

---

## Step 10: Configure Gateway API

```bash
# OIDC Configuration
OIDC_ISSUER_URL="https://dev-12345.okta.com/oauth2/default"
OIDC_AUDIENCE="api://viola-gateway"
AUTH_REQUIRE_BEARER="true"
AUTH_ALLOW_ALGOS="RS256"
OIDC_CLOCK_SKEW_SECONDS="120"

# Rate Limiting
RATE_LIMIT_ENABLED="true"
RATE_LIMIT_PER_MIN_DEFAULT="120"
RATE_LIMIT_KEY_CLAIMS="sub,tid"

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

---

## Step 11: Test Authentication

### Get Access Token (Client Credentials Flow)

```bash
# Service-to-service authentication
curl -X POST \
  https://dev-12345.okta.com/oauth2/default/v1/token \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=client_credentials" \
  -d "client_id=<YOUR_CLIENT_ID>" \
  -d "client_secret=<YOUR_CLIENT_SECRET>" \
  -d "scope=incidents:read alerts:read"
```

### Get Access Token (Authorization Code Flow with PKCE)

For user-based authentication, use the authorization code flow:

```bash
# Step 1: Get authorization code (in browser)
https://dev-12345.okta.com/oauth2/default/v1/authorize?\
  client_id=<YOUR_CLIENT_ID>&\
  response_type=code&\
  scope=openid%20incidents:read%20alerts:read&\
  redirect_uri=https://your-app.com/callback&\
  state=random_state_string&\
  code_challenge=<BASE64_URL_ENCODED_SHA256_HASH>&\
  code_challenge_method=S256

# Step 2: Exchange code for token
curl -X POST \
  https://dev-12345.okta.com/oauth2/default/v1/token \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=authorization_code" \
  -d "client_id=<YOUR_CLIENT_ID>" \
  -d "redirect_uri=https://your-app.com/callback" \
  -d "code=<AUTHORIZATION_CODE>" \
  -d "code_verifier=<RANDOM_STRING_FROM_STEP1>"
```

### Test the API

```bash
# Set token
export TOKEN="<access_token_from_above>"

# List incidents
curl -H "Authorization: Bearer $TOKEN" \
  https://your-gateway.com/api/v1/incidents

# Update incident
curl -X PATCH \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"status": "ack"}' \
  https://your-gateway.com/api/v1/incidents/INC-123
```

---

## Troubleshooting

### Error: "invalid_client: Invalid client credentials"

**Solution:** Double-check Client ID and Client Secret from Step 3.

### Error: "invalid_scope: One or more scopes are not configured"

**Solution:**
- Verify scopes are created in your authorization server (Step 2)
- Ensure scopes are included in your access policy (Step 7)

### JWT Missing `groups` Claim

**Solution:**
- Check the group claim filter regex in Step 6
- Ensure user is assigned to a group matching the pattern
- Verify claim is included in access token (not just ID token)

### JWT Missing `tid` Claim

**Solution:**
- Check that you added the `tenantId` user attribute
- Verify users have the `tenantId` field populated
- Check the claim mapping in Step 6

### MFA Not Working

**Solution:**
- Ensure MFA factors are enabled
- Check sign-on policy is assigned to your app
- Verify policy rule requires MFA
- Check `amr` claim in decoded token

---

## Token Claim Mapping

| Okta Claim | Viola Field | Description |
|------------|-------------|-------------|
| `sub` | Subject | User's unique identifier |
| `email` | Email | User's email address |
| `groups` | Roles | Groups assigned to user |
| `scp` | Scopes | OAuth scopes granted |
| `tid` | TenantID | Custom tenant identifier |
| `iss` | Issuer | Token issuer URL |
| `aud` | Audience | Intended audience |
| `exp` | Expiry | Token expiration time |
| `amr` | - | Authentication methods (for MFA) |

---

## Security Best Practices

1. **Use Authorization Code + PKCE for Web Apps**
   - More secure than implicit flow
   - Protects against authorization code interception

2. **Use Client Credentials for Service-to-Service**
   - No user context needed
   - Simpler authentication flow

3. **Rotate Client Secrets Regularly**
   - Set up secret rotation schedule
   - Use Okta API to automate rotation

4. **Enable MFA for All Users**
   - Require MFA for sensitive operations
   - Use Okta Verify for best experience

5. **Monitor System Log**
   - Okta → Reports → System Log
   - Set up alerts for suspicious activity

6. **Use Short Token Lifetimes**
   - Access token: 1 hour
   - Refresh token: 7 days (or less)

---

## References

- [Okta Developer Documentation](https://developer.okta.com/docs/)
- [Implement OAuth for Okta](https://developer.okta.com/docs/guides/implement-oauth-for-okta/main/)
- [Custom claims](https://developer.okta.com/docs/guides/customize-tokens-returned-from-okta/main/)
- [Sign-on policies](https://help.okta.com/en-us/Content/Topics/Security/policies.htm)

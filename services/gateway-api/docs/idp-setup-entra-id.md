# Microsoft Entra ID (Azure AD) Setup Guide

Complete guide to configure Viola Gateway API with Microsoft Entra ID (formerly Azure Active Directory).

---

## Prerequisites

- Azure account with admin access to Azure Active Directory
- Viola Gateway API deployed and accessible

---

## Step 1: Register Application

1. **Navigate to Azure Portal**
   - Go to https://portal.azure.com
   - Select **Azure Active Directory** from the left menu
   - Click **App registrations** → **New registration**

2. **Configure Application**
   - **Name**: `Viola Gateway API`
   - **Supported account types**: `Accounts in this organizational directory only (Single tenant)`
   - **Redirect URI**: Leave blank (not needed for API-only apps)
   - Click **Register**

3. **Note Your Tenant ID**
   - On the Overview page, copy your **Directory (tenant) ID**
   - Example: `12345678-1234-1234-1234-123456789abc`

---

## Step 2: Expose an API

1. **Set Application ID URI**
   - In your app registration, click **Expose an API**
   - Click **Add** next to "Application ID URI"
   - Use: `api://viola-gateway` (or your preferred identifier)
   - Click **Save**

2. **Add Scopes**

   Click **Add a scope** and create the following scopes:

   | Scope Name | Display Name | Description | Admin Consent | User Consent |
   |------------|--------------|-------------|---------------|--------------|
   | `incidents.read` | Read incidents | View security incidents | Required | Not allowed |
   | `incidents.write` | Write incidents | Update security incidents | Required | Not allowed |
   | `alerts.read` | Read alerts | View security alerts | Required | Not allowed |
   | `alerts.write` | Write alerts | Update security alerts | Required | Not allowed |
   | `rules.read` | Read rules | View detection rules | Required | Not allowed |
   | `rules.write` | Write rules | Manage detection rules | Required | Not allowed |

   For each scope:
   - **Who can consent**: Admins and users (or Admins only for sensitive scopes)
   - **State**: Enabled

---

## Step 3: Add App Roles (Recommended)

1. **Navigate to App Roles**
   - Click **App roles** → **Create app role**

2. **Create Roles**

   | Display Name | Value | Description | Allowed Member Types |
   |--------------|-------|-------------|---------------------|
   | SOC Reader | `SOCReader` | View-only access to incidents and alerts | Users/Groups |
   | SOC Responder | `SOCResponder` | Triage and respond to incidents | Users/Groups |
   | SOC Engineer | `SOCEngineer` | Create and manage detection rules | Users/Groups |
   | Viola Admin | `ViolaAdmin` | Full administrative access | Users/Groups |

   For each role:
   - **Allowed member types**: Users/Groups
   - **Value**: Must match exactly (case-sensitive)
   - **Do you want to enable this app role**: Yes

---

## Step 4: Assign Users/Groups

1. **Navigate to Enterprise Applications**
   - Go to **Azure Active Directory** → **Enterprise applications**
   - Search for and select `Viola Gateway API`

2. **Assign Users**
   - Click **Users and groups** → **Add user/group**
   - Select users or groups
   - Assign appropriate role (SOCReader, SOCResponder, etc.)
   - Click **Assign**

---

## Step 5: Configure Token Claims

### Add Custom Claims (Optional)

1. **Navigate to Token Configuration**
   - In your app registration, click **Token configuration**
   - Click **Add optional claim**

2. **Add Access Token Claims**
   - Token type: **Access**
   - Select claims:
     - `email`
     - `preferred_username`
     - `upn`
   - Check "Turn on the Microsoft Graph email, profile permission"
   - Click **Add**

### Add Custom Tenant Claim (Important for Multi-Tenant)

1. **Create Custom Attribute (if needed)**
   - Go to **Azure Active Directory** → **Users**
   - Click **User settings** → **Manage extension attributes**
   - Add custom attribute: `tenantId` (type: String)

2. **Map Claim in Token**
   - This requires a custom claims mapping policy
   - See: https://learn.microsoft.com/en-us/azure/active-directory/develop/active-directory-claims-mapping

---

## Step 6: Get Configuration Values

### Issuer URL
```
https://login.microsoftonline.com/<TENANT_ID>/v2.0
```

Replace `<TENANT_ID>` with your Directory (tenant) ID from Step 1.

**Example:**
```
https://login.microsoftonline.com/12345678-1234-1234-1234-123456789abc/v2.0
```

### Audience
```
api://viola-gateway
```

(Or whatever you set in Step 2)

### JWKS URL (Auto-discovered)
The gateway will automatically discover this from:
```
https://login.microsoftonline.com/<TENANT_ID>/v2.0/.well-known/openid-configuration
```

**Discovered JWKS URL:**
```
https://login.microsoftonline.com/<TENANT_ID>/discovery/v2.0/keys
```

---

## Step 7: Configure Gateway API

Create a `.env` file or set environment variables:

```bash
# OIDC Configuration
OIDC_ISSUER_URL="https://login.microsoftonline.com/<YOUR_TENANT_ID>/v2.0"
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
SERVICE_NAME="gateway-api"

# Database
PG_HOST="localhost"
PG_PORT="5432"
PG_USER="postgres"
PG_PASSWORD="<your-password>"
PG_DATABASE="viola_gateway"
PG_SSLMODE="require"

# Server
PORT="8080"
VIOLA_ENV="prod"
```

---

## Step 8: Test Authentication

### Get an Access Token

#### Option 1: Using Azure CLI
```bash
# Login
az login

# Get token
TOKEN=$(az account get-access-token --resource api://viola-gateway --query accessToken -o tsv)

echo $TOKEN
```

#### Option 2: Using Postman
1. Create new request
2. Authorization → Type: OAuth 2.0
3. Configure:
   - Grant Type: `Authorization Code` or `Client Credentials`
   - Auth URL: `https://login.microsoftonline.com/<TENANT_ID>/oauth2/v2.0/authorize`
   - Access Token URL: `https://login.microsoftonline.com/<TENANT_ID>/oauth2/v2.0/token`
   - Client ID: Your app's Client ID
   - Scope: `api://viola-gateway/incidents.read`
4. Click **Get New Access Token**

#### Option 3: Using curl (Client Credentials Flow)
```bash
# For service-to-service authentication
curl -X POST \
  https://login.microsoftonline.com/<TENANT_ID>/oauth2/v2.0/token \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=client_credentials" \
  -d "client_id=<YOUR_CLIENT_ID>" \
  -d "client_secret=<YOUR_CLIENT_SECRET>" \
  -d "scope=api://viola-gateway/.default"
```

### Test the API

```bash
# List incidents
curl -H "Authorization: Bearer $TOKEN" \
  https://your-gateway.com/api/v1/incidents

# Get specific incident
curl -H "Authorization: Bearer $TOKEN" \
  https://your-gateway.com/api/v1/incidents/INC-123

# Update incident status
curl -X PATCH \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"status": "ack"}' \
  https://your-gateway.com/api/v1/incidents/INC-123
```

---

## Step 9: Enable MFA (Optional but Recommended)

### Configure Conditional Access

1. **Create Conditional Access Policy**
   - Go to **Azure Active Directory** → **Security** → **Conditional Access**
   - Click **New policy**

2. **Configure Policy**
   - **Name**: `Require MFA for Viola API`
   - **Users**: Select users/groups who need MFA
   - **Cloud apps**: Select `Viola Gateway API`
   - **Conditions**: Configure as needed (e.g., location, device)
   - **Grant**: Require multi-factor authentication
   - **Enable policy**: On

3. **Result**
   - When users authenticate, the `amr` claim will include `mfa`
   - Or the `acr` claim will be set to indicate MFA was used

---

## Step 10: Configure Token Introspection (Optional)

For real-time token revocation checks:

1. **Create Client Credentials**
   - In your app registration, go to **Certificates & secrets**
   - Click **New client secret**
   - Description: `Token Introspection`
   - Expires: Choose appropriate duration
   - Click **Add** and copy the secret value

2. **Configure Gateway**
   ```bash
   INTROSPECTION_ENABLED="true"
   INTROSPECTION_ENDPOINT="https://login.microsoftonline.com/<TENANT_ID>/oauth2/v2.0/introspect"
   INTROSPECTION_CLIENT_ID="<YOUR_CLIENT_ID>"
   INTROSPECTION_CLIENT_SECRET="<YOUR_CLIENT_SECRET>"
   ```

---

## Troubleshooting

### Error: "AADSTS50105: The signed in user is not assigned to a role"

**Solution:** Assign the user to an app role (see Step 4).

### Error: "AADSTS700016: Application not found"

**Solution:** Check your `OIDC_AUDIENCE` matches the Application ID URI from Step 2.

### Error: "invalid_token: Audience validation failed"

**Solution:** Ensure the token's `aud` claim matches `OIDC_AUDIENCE` exactly.

### JWT Missing `roles` Claim

**Solution:**
- Ensure user is assigned an app role
- Check **Token configuration** includes the `roles` claim
- Use access token, not ID token

### JWT Missing `tid` (Tenant ID) Claim

**Solution:**
- Add a custom claims mapping policy
- Or extract tenant from the `iss` claim (issuer URL contains tenant ID)

---

## Token Claim Mapping

Here's how Entra ID JWT claims map to Viola's internal fields:

| Entra ID Claim | Viola Field | Description |
|----------------|-------------|-------------|
| `sub` | Subject | User's unique identifier |
| `oid` | Subject | Object ID (alternative) |
| `email` / `upn` | Email | User's email address |
| `roles` | Roles | App roles assigned to user |
| `scp` | Scopes | OAuth scopes granted |
| `tid` | TenantID | Azure AD tenant ID |
| `iss` | Issuer | Token issuer URL |
| `aud` | Audience | Intended audience |
| `exp` | Expiry | Token expiration time |
| `amr` | - | Authentication methods (for MFA) |
| `acr` | - | Authentication context (for MFA) |

---

## Security Best Practices

1. **Use Application Permissions for Service Accounts**
   - Grant type: `client_credentials`
   - Scope: `api://viola-gateway/.default`

2. **Use Delegated Permissions for User Access**
   - Grant type: `authorization_code`
   - Scopes: Specific scopes like `incidents.read`

3. **Enable Conditional Access**
   - Require MFA for sensitive operations
   - Restrict access by location/device

4. **Rotate Client Secrets Regularly**
   - Set expiration on secrets
   - Use Azure Key Vault for secret storage

5. **Monitor Sign-In Logs**
   - Azure AD → Monitoring → Sign-in logs
   - Set up alerts for suspicious activity

6. **Use Managed Identities (if deployed on Azure)**
   - Avoid storing credentials in code
   - Use system-assigned or user-assigned identities

---

## References

- [Microsoft identity platform documentation](https://learn.microsoft.com/en-us/azure/active-directory/develop/)
- [Protect an API using OAuth 2.0](https://learn.microsoft.com/en-us/azure/active-directory/develop/scenario-protected-web-api-overview)
- [Configure app roles](https://learn.microsoft.com/en-us/azure/active-directory/develop/howto-add-app-roles-in-azure-ad-apps)
- [Conditional Access](https://learn.microsoft.com/en-us/azure/active-directory/conditional-access/)

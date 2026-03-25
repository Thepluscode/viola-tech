#!/usr/bin/env node
/**
 * Fetches a dev JWT from the Viola auth service (http://localhost:8081/token).
 *
 * Usage:
 *   node scripts/get-token.mjs
 *   node scripts/get-token.mjs --role admin --tid my-tenant
 *
 * Then copy the printed token into .env.local as DEV_API_TOKEN=<token>
 * and set NEXT_PUBLIC_USE_MOCK=false.
 */

const args = process.argv.slice(2);
function getArg(flag, fallback) {
  const idx = args.indexOf(flag);
  return idx !== -1 ? args[idx + 1] : fallback;
}

const body = {
  sub:   getArg("--sub",   "dev-user"),
  tid:   getArg("--tid",   "dev-tenant"),
  email: getArg("--email", "analyst@viola.corp"),
  role:  getArg("--role",  "analyst"),
};

const AUTH_URL = "http://localhost:8081/token";

try {
  const res = await fetch(AUTH_URL, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });

  if (!res.ok) {
    const text = await res.text();
    console.error(`Auth service error ${res.status}: ${text}`);
    process.exit(1);
  }

  const data = await res.json();
  const token = data.access_token;

  console.log("\n✓ Token issued for:", JSON.stringify(body));
  console.log("\nAdd to .env.local:\n");
  console.log(`NEXT_PUBLIC_USE_MOCK=false`);
  console.log(`DEV_API_TOKEN=${token}`);
  console.log("\nToken expires in:", data.expires_in, "seconds\n");
} catch (err) {
  console.error("Failed to reach auth service at", AUTH_URL);
  console.error("Make sure the backend is running: cd ../../ && make dev");
  console.error(err.message);
  process.exit(1);
}

# Pocket ID OAuth MCP demo

This demo runs [Pocket ID](https://github.com/pocket-id/pocket-id) and an OAuth-protected MCP server with Docker Compose. Two [Cloudflare quick tunnels](https://developers.cloudflare.com/cloudflare-one/connections/connect-networks/do-more-with-tunnels/trycloudflare/) provide temporary public HTTPS URLs for Pocket ID and the MCP server.

The MCP server provides the following tools:

- `list_notes` returns the notes for the authenticated user. It requires the `notes:read` scope.
- `add_note` adds a note for the authenticated user. It requires the `notes:write` scope.
- `clear_notes` removes all notes for the authenticated user. It requires the `notes:write` scope.

## Before you begin

You need the following software:

- Docker with Docker Compose
- [Claude Desktop](https://claude.ai/desktop) or [Claude Code](https://claude.ai/code)

## Run the demo

### 1. Start the tunnels

The demo needs public HTTPS URLs for Pocket ID so we use Cloudflare tunnels.

Start the Cloudflare tunnel containers:

```sh
docker compose up -d pocket-id-tunnel mcp-tunnel
```

The tunnel containers generate new public URLs when they start. Get the URLs from the container logs:

```sh
echo "Pocket ID URL:"
docker compose logs pocket-id-tunnel | grep -oE 'https://[^ ]+\.trycloudflare\.com'
echo "MCP URL:"
docker compose logs mcp-tunnel | grep -oE 'https://[^ ]+\.trycloudflare\.com'
```

### 2. Configure the services

Copy the example environment file:

```sh
cp .env.example .env
```

... and replace the placeholders with the public URLs from the tunnel logs. The file should look like this:

```dotenv
POCKET_ID_URL=https://<pocket-id-tunnel>.trycloudflare.com
MCP_URL=https://<mcp-tunnel>.trycloudflare.com/mcp
POCKET_ID_ENCRYPTION_KEY=<generated-encryption-key>
```

Start Pocket ID and the MCP server:

```sh
docker compose up -d
```

### 3. Configure Pocket ID

1. Open `<POCKET_ID_URL>/setup`.
2. Create the administrator account and register a passkey.
3. Open **Administration → APIs**.
4. Select **Add API**.
5. Configure the API:

   | Field        | Value                                             |
   | ------------ | ------------------------------------------------- |
   | Name         | `Notes MCP Server`                                |
   | API resource | The exact `MCP_URL` from `.env`, including `/mcp` |

6. Add the following API permissions:

   | Key           | Name                |
   | ------------- | ------------------- |
   | `notes:read`  | `Read your notes`   |
   | `notes:write` | `Manage your notes` |

7. In the **Access** card, select **Metadata document clients**.
8. Enable **Allow all metadata document clients**.
9. Select `notes:read` and `notes:write`.
10. Select **Save**.

The Compose configuration allows the Claude Code and Claude Desktop client metadata URLs through Pocket ID's `CIMD_URL_ALLOWLIST`. You do not need to create an OIDC client, a client secret, or a client-access rule.

### 4. Connect Claude Code

Add the MCP server and start the sign-in flow:

```sh
claude mcp add --transport http demo <MCP_URL>
claude mcp login demo
```

Replace `<MCP_URL>` with the complete value from `.env`.

After you sign in through Pocket ID, test the connection with the following prompt:

```text
Add a note that says the Pocket ID OAuth demo works.
```

### 5. Connect Claude Desktop

1. Open Claude Desktop.
2. Go to **Settings → Connectors**.
3. Select **Add → Add custom connector**.
4. Enter a name and the complete `MCP_URL` from `.env`.
5. Select **Add**.
6. Select **Connect**, and then sign in through Pocket ID.

Test the connection with the following prompt:

```text
Add a note that says the Pocket ID OAuth demo works.
```

## Stop the demo

Stop the containers:

```sh
docker compose down
```

Cloudflare assigns new tunnel URLs the next time the tunnel containers are created. If the URLs change, update `.env` and the API resource in Pocket ID before restarting the services.

## Add authentication to your MCP server

Setting up an OAuth 2.0 provider is often complex and time-consuming. We built Pocket ID because authentication should not be the hard part of building an MCP server or APIs. We can confidently say that Pocket ID is the easiest OAuth 2.0 provider to set up and use.

### Deploy Pocket ID

Setting up Pocket ID is easy. In fact, you have just set up your own instance. For production, deploy it on a server and place a reverse proxy in front of it.

Follow the [Pocket ID installation guide](https://pocket-id.org/docs/setup/installation) for the production setup.

### What do you need to do in your MCP server?

The short version is: verify the JWT.

Pocket ID issues JWT access tokens to clients such as Claude Code. The clients use these tokens to authenticate to your MCP server.

Use a maintained JWT or OpenID Connect library to validate each token. You can find a Go implementation in [`mcp-server/auth.go`](mcp-server/auth.go). Your library must perform the following checks:

1. Verify the token signature with a key from Pocket ID's JWKS endpoint at `/.well-known/jwks.json`.
2. Allow only the configured signing algorithms.
3. Verify that the `iss` claim matches the Pocket ID URL.
4. Verify that the `aud` claim matches the MCP resource URL.
5. Verify that the token has not expired.
6. Require a non-empty `sub` claim.

Configure the library with the Pocket ID issuer, the MCP resource URL, and the Pocket ID JWKS endpoint. Do not implement JWT parsing or signature verification yourself. There is a library for that.

Inside your tools, check the `scope` claim before performing an action. This demo requires `notes:read` for `list_notes` and `notes:write` for `add_note` and `clear_notes`. See [`mcp-server/scopes.go`](mcp-server/scopes.go) for the scope checks.

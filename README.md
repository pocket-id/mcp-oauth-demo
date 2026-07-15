# Pocket ID OAuth MCP demo

With [Pocket ID](https://github.com/pocket-id/pocket-id), you can protect your MCP server with OAuth 2.1 and OpenID Connect. This demo shows how quickly OAuth can be added to an MCP server with Pocket ID.

The Docker Compose stack contains:

- **Open WebUI**, available at <http://localhost:3067>.
- **Demo Notes MCP**, a small Go server available inside Docker at `http://mcp:8080/mcp`.

You will create a temporary Pocket ID instance at [demo.pocket-id.org](https://demo.pocket-id.org). The instance is available for 1 hour and is deleted automatically afterward.

The MCP server exposes three tools:

- `add_note` stores a note for the authenticated user.
- `list_notes` lists only that user's notes.
- `clear_notes` deletes only that user's notes.

Notes exist only in memory and disappear when the MCP container restarts. Notes are separated by the OAuth access token's `sub` claim, which makes it easy to demonstrate that two Pocket ID users do not see each other's data.

## What the demo proves

During the demo:

1. Open WebUI discovers the MCP server's OAuth metadata.
2. Open WebUI redirects the user to the temporary Pocket ID instance.
3. Pocket ID asks the user to approve `notes:read` and `notes:write`.
4. Pocket ID issues an access token for the MCP resource `http://mcp:8080/mcp`.
5. The MCP server verifies the token's signature, issuer, audience, expiry, subject, and scopes before running a tool.

## Prerequisites

You need:

- Docker with Docker Compose.
- 1 hour to complete the demo before the temporary Pocket ID instance expires. No worries, with Pocket ID it's so easy that you don't even need 30 minutes.

## Step 1: Create a temporary Pocket ID instance

1. Open [demo.pocket-id.org](https://demo.pocket-id.org).
2. Click **Start Demo**.
3. Complete the Pocket ID setup and register your passkey.
4. Copy the instance's base URL without `/setup`. For example:

   ```text
   https://<demo-id>.demo.pocket-id.org
   ```

This guide calls that value `<POCKET_ID_URL>`.

## Step 2: Prepare the demo environment

Copy the example environment file:

```sh
cp .env.example .env
```

and replace both occurrences of `replace-with-your-demo-id` with the ID from your Pocket ID demo URL.

## Step 3: Start Open WebUI and the MCP server

Build and start the stack:

```sh
docker compose up --build -d
```

Check that both services are running:

```sh
docker compose ps
```

Open <http://localhost:3067> and create the initial Open WebUI account if this is the first run. The first account becomes the administrator.

## Step 4: Register the MCP API in Pocket ID

Return to your `<POCKET_ID_URL>` tab and sign in as the administrator.

1. Open **Settings → Administration → APIs**.
2. Click **Add API**.
3. Enter:

   | Field        | Value                 |
   | ------------ | --------------------- |
   | Name         | `Demo Notes MCP`      |
   | API resource | `http://mcp:8080/mcp` |

4. Save the API.
5. In the API's **API permissions** section, add these two permissions:

   | Key           | Name          | Suggested description                               |
   | ------------- | ------------- | --------------------------------------------------- |
   | `notes:read`  | `Read notes`  | `Read the signed-in user's demo notes`              |
   | `notes:write` | `Write notes` | `Create and delete the signed-in user's demo notes` |

6. Save the permissions.

The API resource must be exactly `http://mcp:8080/mcp`. Although `mcp` is a Docker-internal hostname, this value is an OAuth audience identifier. Pocket ID does not need to connect to it.

## Step 5: Register Open WebUI as an OIDC client

The Open WebUI tool-server ID used in this guide is `notes`. Open WebUI includes that ID in its OAuth callback URL, so configure the callback exactly as shown below.

1. In Pocket ID, open **Settings → Administration → OIDC Clients**.
2. Click **Add OIDC Client**.
3. Enter:

   | Field        | Value                                                    |
   | ------------ | -------------------------------------------------------- |
   | Name         | `Open WebUI MCP`                                         |
   | Callback URL | `http://localhost:3067/oauth/clients/mcp:notes/callback` |

4. Save the client.
5. Copy its **Client ID** and **Client Secret**. You will enter both in Open WebUI.
6. Open the client's **API Access** tab.
7. Find `Demo Notes MCP` and click **Edit**.
8. Under **User-delegated access**, select both:
   - `notes:read`
   - `notes:write`
9. Leave **Client access** unselected. This demo acts on behalf of a signed-in user and does not use the client-credentials grant.
10. Save the API access settings.

## Step 6: Add the MCP server to Open WebUI

Open Open WebUI at <http://localhost:3067> and sign in as its administrator.

1. Open **Admin Panel → Integrations**.
2. Add a new **External Tool Server**.
3. Click **OpenAPI** to switch to the MCP tool type.
4. Configure the general fields:

   | Field | Value                 |
   | ----- | --------------------- |
   | ID    | `notes`               |
   | Name  | `OAuth Notes Demo`    |
   | URL   | `http://mcp:8080/mcp` |

5. Configure authentication:

   | Field                    | Value                                                                      |
   | ------------------------ | -------------------------------------------------------------------------- |
   | Authentication           | `OAuth 2.1 (Static)`                                                       |
   | Client ID                | Client ID copied from Pocket ID                                            |
   | Client Secret            | Client secret copied from Pocket ID                                        |
   | OAuth Server URL         | Your `<POCKET_ID_URL>`, for example `https://<demo-id>.demo.pocket-id.org` |
   | OAuth Resource Parameter | `Automatic`                                                                |
   | OAuth Scopes             | `Use discovered scopes`                                                    |

6. Click **Register Client**.
7. Save the external-tool connection.

Do not change the Open WebUI ID from `notes` after registering the client. If you use another ID, replace `notes` in the Pocket ID callback URL with that ID before authorizing.

## Step 7: Authorize and use the tool

OAuth MCP tools must be enabled interactively before the model calls them.

1. Start a new chat in Open WebUI.
2. Use the **Integrations** icon in the chat input.
3. Click **Tools** and enable `OAuth Notes Demo`.
4. Open WebUI redirects you to your temporary Pocket ID instance.
5. Sign in and approve the requested `notes:read` and `notes:write` permissions.
6. Pocket ID redirects you to Open WebUI.

Try these prompts in order:

```text
Use the notes tool to remember that the Pocket ID OAuth demo works.
```

```text
List all of my demo notes.
```

```text
Clear all of my demo notes.
```

The model should use `add_note`, `list_notes`, and `clear_notes` respectively.

## Step 8: Demonstrate user isolation

For a more visible OAuth demonstration:

1. Authorize the tool as one Pocket ID user and add a note.
2. Sign in to Open WebUI as a different user.
3. Enable the MCP tool and authorize through Pocket ID as the second user.
4. List the notes.

The second user receives an empty list because the MCP server stores notes separately for each OAuth subject.

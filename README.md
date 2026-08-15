# 🌌 Nexus

**The Open-Source Identity, Access, and Infrastructure Provisioning Hub**

Nexus is a centralized, self-hosted controller designed to automate user onboarding, manage SSH/system access across edge environments, and dynamically provision infrastructure. It combines a secure Go backend, an ultra-fast Svelte frontend, and real-time WebSocket Edge Agents into a single, cohesive platform.

---

## ✨ Core Features

### 🔐 User Lifecycle & Onboarding

* **Secure Token Onboarding:** Users receive secure, expiring invite links to complete their profiles.
* **Dynamic Form Generation:** Nexus dynamically asks users for required variables (like SSH Public Keys or Passwords) based on the infrastructure they are assigned to.
* **Automated Expiration & Deprovisioning:** Background cron jobs track user expiration dates, automatically firing soft or hard deprovisioning pipelines when access expires.

### 🖥️ Edge Agent Infrastructure

* **Real-time WebSockets:** Nexus Core communicates with remote Linux Edge Agents via secure WebSockets.
* **Pipeline Macros:** Define provisioning pipelines (e.g., `useradd`, `mkdir .ssh`, `uv init`) using simple JSON/YAML structures.
* **Lifecycle Binding:** Bind specific Macros to an Edge Agent for Onboarding, Soft Deprovisioning, and Destructive (Hard) Deprovisioning.

### 📝 Dynamic Markdown CMS

* **Built-in Documentation:** Write and publish SSH Quickstart guides, Welcome Emails, and Public Docs directly in the Nexus Dashboard.
* **Contextual Variable Injection:** Uses Go's native template engine to inject user-specific context into markdown dynamically.
* *Example:* `ssh {{.Username}}@{{.Address}}` automatically renders the correct IP and Username for the person reading the document.


* **Array Support:** Loop through a user's assigned servers using `{{range .Servers}}`.

### ⚡ Modern UX/UI

* **Persistent Dashboard:** Client-side view memory using `sessionStorage`.
* **Live Polling:** Background polling keeps Agent statuses (`ONLINE`/`OFFLINE`) and Pipeline Execution Logs up-to-date in real time.
* **Theming:** Fully themable interface built on TailwindCSS and DaisyUI.

---

## 🏗️ Architecture

Nexus relies on a dual-execution model:

1. **Nexus Core (Central Hub):** The main web server, UI, and SQLite database. It manages state, templates, and API routes.
2. **Edge Agents (Remote Nodes):** Lightweight Go binaries installed on your target servers. They connect *back* to Nexus Core, listen for tasks, and execute local shell commands (like creating OS users or writing SSH keys).

---

## 🚀 Getting Started

Nexus is designed to run anywhere using a multi-stage Docker build, packing both the Svelte frontend and Go backend into a single, lightweight Alpine Linux container.

### Running with Docker

```bash
# Clone the repository
git clone https://github.com/your-username/nexus.git
cd nexus

# Build the unified container
docker build -t nexus-server .

# Run the container (exposing port 8080 and mounting the SQLite data volume)
docker run -d \
  -p 8080:8080 \
  -v nexus_data:/app/data \
  --name nexus \
  nexus-server

```

---

## 🚧 Work in Progress: Declarative Plugin System

> **Note:** This feature is currently in active development.

As Nexus evolves from a "script runner" to a complete **Identity Hub**, we are introducing a **Declarative Plugin Architecture**. This allows Nexus to interact with external REST APIs (like Gitea, Slack, Cloudflare) *without* needing to deploy an Edge Agent or write custom Go code.

### The Vision

Plugins are strict YAML manifests downloaded from an external Plugin Catalog. They cannot execute arbitrary code, making them highly secure.

* **API Execution:** Map Nexus variables to secure HTTP requests natively.
* **Providers:** Plugins can seamlessly hook into core Nexus features:
* `avatar_provider`: Resolve profile pictures (e.g., via Gitea or Gravatar).
* `pages_provider`: Auto-inject quickstart documentation into the CMS upon installation.
* `variable_provider`: Make new variables available to the Markdown engine (e.g., `{{.plugin.gitea.url}}`).


* **Secret Management:** Plugin manifests define their required secrets, which Nexus automatically encrypts, manages, and injects securely at runtime.

#### Example Plugin Architecture (Future):

```yaml
id: "nexus-slack-alerts"
name: "Slack Notifications"
config:
  - key: "slack_bot_token"
    type: "secret"
provides:
  - "notification_provider"
capabilities:
  send_welcome_message:
    action:
      type: "http_request"
      method: "POST"
      url: "https://slack.com/api/chat.postMessage"

```

---

## 🛡️ Security

* **No Turing-Complete Plugins:** The upcoming plugin system relies purely on Declarative YAML and a sandboxed HTTP engine, preventing malicious OS-level execution.
* **Strict Token Lifecycles:** Invite links are hashed via `bcrypt` and aggressively timed out.
* **Session Storage:** JWTs are stored in ephemeral `sessionStorage` to prevent unauthorized cross-session access.


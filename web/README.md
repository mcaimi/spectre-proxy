# SPECTRE Web UI

Frontend for **SPECTRE** — **S**ecure **P**rotocol **E**xamination and **C**apture **T**ool for **R**everse **E**ngineering. An HTTP/HTTPS interception proxy with a React-based management dashboard.

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        Browser                                 │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │                    React 18 App                            │  │
│  │  ┌──────────┐ ┌──────────────┐ ┌──────────────┐           │  │
│  │  │ Dashboard│ │ VHost List   │ │ Certificate  │           │  │
│  │  │ Component│ │ Component    │ │ Manager      │           │  │
│  │  └──────────┘ └──────────────┘ └──────────────┘           │  │
│  │  ┌───────────────────────────────────────────────┐         │  │
│  │  │          Request Log Viewer                    │         │  │
│  │  └───────────────────────────────────────────────┘         │  │
│  │                                                             │  │
│  │  Router ─→ axios API layer ─→ /api/v1 ─→ backend proxy     │  │
│  └───────────────────────────────────────────────────────────┘  │
│                          │                                      │
│                  Vite dev proxy :3000                           │
│                  ↓ (forward /api → :9000)                      │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │              SPECTRE Backend (Go)                          │  │
│  │              localhost:9000 /api/v1                        │  │
│  └───────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
```

### Technology Stack

| Layer        | Technology                          |
|-------------|-------------------------------------|
| Framework   | React 18 with JS(X)                 |
| Bundler     | Vite 5                              |
| Routing     | React Router DOM v6                  |
| HTTP Client | Axios                               |
| Styling     | Custom CSS (app.css + index.css)     |

## Project Structure

```
web/ui/
├── package.json              # Dependencies and build scripts
├── vite.config.js            # Vite config (dev proxy → backend :9000)
└── src/
    ├── main.jsx              # Entry point — mounts <App> under React.StrictMode
    ├── App.jsx               # Root layout with router, navbar, and routes
    ├── App.css               # App-level styles (navbar, navbar, footer, cards)
    ├── index.css             # Global styles (body, tables, badges, modal)
    ├── api.js                # Axios API layer — typed endpoint modules
    └── components/
        ├── Dashboard.jsx     # Overview stats + recent requests (5s auto-refresh)
        ├── VHostList.jsx     # CRUD for virtual host routing rules
        ├── CertList.jsx      # SSL cert management, CA download & replacement
        └── RequestLog.jsx    # Paginated request log viewer with detail modal
```

## Component Hierarchy

```
App (Router, Navbar, Footer)
├── Dashboard
│   ├── StatsGrid (4 stat cards)
│   ├── RecentRequestsTable
│   └── QuickStart guide
├── VHostList
│   ├── VHostTable
│   ├── Add/Edit Modal (form)
│   └── Toggle enabled/disabled
├── CertList
│   ├── CertificateTable (with expiry badges)
│   ├── Download Root CA
│   └── Replace CA form (full X.509 fields)
└── RequestLog
    ├── LogTable (paginated)
    ├── Detail Modal (request/response headers & bodies)
    └── Auto-refresh toggle
```

## Route Map

| Route          | Component        | Description                                    |
|---------------|------------------|------------------------------------------------|
| `/`           | `Dashboard`      | Overview stats, recent logs, quick start guide |
| `/vhosts`     | `VHostList`      | Create, edit, delete, enable/disable vhosts     |
| `/certificates` | `CertList`    | View, delete certs; download/replace Root CA    |
| `/logs`       | `RequestLog`     | Paginated request log viewer with detail modal  |

## API Layer

The API client (`src/api.js`) wraps Axios into a namespaced module system, one module per resource:

| Module      | Endpoints                              | Operation                  |
|-------------|----------------------------------------|----------------------------|
| `healthAPI` | `GET /health`                          | Health check               |
| `vhostAPI`  | `GET /list`, `CREATE`, `PUT /:id`, `DELETE /:id` | Virtual host CRUD |
| `certAPI`   | `GET /list`, `DELETE /:id`, `GET /ca`, `POST /ca/replace` | Cert mgmt    |
| `logAPI`    | `GET /list?limit=&offset=`, `GET /:id` | Request log retrieval       |

All requests go to `/api/v1` prefix, which is proxied to the Go backend (`localhost:9000`) in development via Vite's dev server proxy.

## Component Details

### Dashboard
- Loads stats and the latest 10 request logs via `Promise.all` parallel calls on mount.
- Auto-refreshes dashboard data every 5 seconds using `setInterval`.
- Displays four stat cards: total virtual hosts, active hosts, certificates, total log count.
- Shows an onboarding warning when no virtual hosts exist, with curl examples.
- Provides a quick-start section with certificate installation commands per platform.

### VHostList
- Fetches all virtual hosts from `GET /api/v1/vhosts`.
- Renders an editable table with hostname, target URL, enabled status, and created date.
- Toggle enabled/disabled inline with a single click.
- Modal form for create/edit with hostname (supports wildcards like `*.example.com`) and target URL fields.
- Deletion guarded by browser `confirm()` dialog.

### CertList
- Lists all generated TLS certificates with computed expiry badges (expired / expiring-soon / valid).
- Download the Root CA directly via `window.location.href` (triggers browser download).
- Replace the Root CA with a full X.509 subject form: CN, O, OU, C, ST, L, validity years.
- Inline installation instructions for macOS, Linux, and Windows.
- "How It Works" section explaining on-demand MITM certificate generation flow.

### RequestLog
- Fetches paginated logs with configurable page size (default 50).
- Auto-refresh every 3 seconds (toggleable).
- Navigation between pages with Previous/Next buttons and disabled state management.
- Detail modal showing full request/response breakdown: method, URL, headers, and body.
- Binary request/response bodies are base64-encoded in transit; decoded client-side with `atob()`.
- JSON bodies are pretty-printed automatically.

## State Management

All components use **local React state** via `useState` and `useEffect` hooks. There is no global state management library (no Redux, Zustand, etc.). Each component manages its own:

- **Data state**: loaded from the backend (`vhosts`, `certs`, `logs`, `stats`)
- **UI state**: modals, loading spinners, error messages, pagination
- **Form state**: controlled inputs for create/edit operations

Data fetching strategy:
- **Dashboard**: `Promise.all` parallel requests on mount + 5-second interval
- **VHostList/CertList**: single fetch on mount, refresh after mutations
- **RequestLog**: fetch on mount + pagination changes + 3-second interval

## Development

```bash
cd web/ui
npm install
npm run dev    # starts Vite dev server on :3000, proxies /api → :9000
npm run build  # production build → dist/
npm run preview # preview production build locally
```

The Vite dev server at `:3000` proxies all `/api` requests to the SPECTRE backend at `localhost:9000` (configured in `vite.config.js`).

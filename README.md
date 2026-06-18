# SPECTRE

**S**ecure **P**rotocol **E**xamination and **C**apture **T**ool for **R**everse **E**ngineering

A transparent HTTP/HTTPS interception proxy for reverse engineers and security professionals.

## Features

- **Transparent MITM Proxy**: Intercepts HTTP and HTTPS traffic with minimal client configuration
- **On-Demand Certificate Generation**: Automatically generates SSL certificates for intercepted domains using SNI
- **Custom CA Certificate**: Replace the default CA with your own custom certificate fields
- **Virtual Host Management**: Route multiple domains through the proxy with independent configurations
- **Request/Response Logging**: Captures and stores all intercepted traffic in SQLite database
- **Modern Web UI**: React-based management interface for visual control
- **REST API**: Complete API for programmatic management
- **Zero Configuration**: SQLite-based storage requires no external dependencies

## Quick Start

### Installation

```bash
# Build the binary (API-only, includes embedded UI placeholder)
make build

# Or build with full React web UI (requires Node.js 18+)
make build-full
```

**Note**:

- `make build` - Works without Node.js, includes enhanced HTML placeholder with API docs
- `make build-full` - Requires Node.js 18+, includes full React dashboard
- After `make clean`, `make build` will still work (placeholder is protected)

#### Option 2: Podman

```bash
# prepare a valid spectre configuration file
cp spectre.yaml.example spectre.yaml

# Build the container image
podman build -t spectre:latest -f containers/Containerfile .

# Quick start with Podman
podman run -d \
  --name spectre \
  -p 8080:8080 \
  -p 8443:8443 \
  -p 9000:9000 \
  -v spectre-data:/data \
  spectre:latest

# Access UI
open http://localhost:9000/
```

### Running SPECTRE

#### Native

```bash
# Run with default configuration
./spectre

# Run with custom config file
./spectre --config /path/to/spectre.yaml
```

### Configuration

Create a `spectre.yaml` file (see `spectre.yaml.example` for reference):

```yaml
proxy:
  httpport: 8080
  httpsport: 8443
  bindaddr: "0.0.0.0"
  readtimeout: 30s
  writetimeout: 30s
  idletimeout: 120s
  maxconnections: 1000

webui:
  enabled: true
  port: 9000
  bindaddr: "127.0.0.1"

database:
  path: "./spectre.db"
  maxopenconns: 25
  maxidleconns: 5

tls:
  certcachesize: 100
  cakeysize: 4096
  certkeysize: 2048
  certvalidity: 8760h

logging:
  level: "info"
  format: "text"
  output: "stdout"
```

## Usage

### 1. Start the Proxy

```bash
./spectre
```

The proxy will:

- Listen for HTTP traffic on port 8080
- Listen for HTTPS traffic on port 8443
- Start the management API on port 9000
- Generate a Root CA certificate on first run

### 2. Install Root CA Certificate

To intercept HTTPS traffic, install the Root CA certificate in your system's trust store:

```bash
# Download the CA certificate
curl http://localhost:9000/api/v1/certificates/ca > spectre-ca.crt

# macOS
sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain spectre-ca.crt

# Linux (Ubuntu/Debian)
sudo cp spectre-ca.crt /usr/local/share/ca-certificates/
sudo update-ca-certificates

# Windows
certutil -addstore -f "ROOT" spectre-ca.crt
```

### 3. Add a Virtual Host

```bash
# Add a virtual host to intercept api.example.com
curl -X POST http://localhost:9000/api/v1/vhosts \
  -H "Content-Type: application/json" \
  -d '{
    "hostname": "api.example.com",
    "target_url": "https://real-api.example.com"
  }'
```

### 4. Route Traffic to Proxy

Add an entry to `/etc/hosts`:

```
127.0.0.1 api.example.com
```

### 5. Intercept Traffic

```bash
# Make a request through the proxy
curl https://api.example.com/endpoint

# View captured requests via API
curl http://localhost:9000/api/v1/logs

# Or use the Web UI
open http://localhost:9000/
```

## Web UI

SPECTRE includes a modern React-based web interface for easy management.

### Accessing the UI

Once SPECTRE is running, open your browser to:

```
http://localhost:9000/
```

### Features

**Dashboard**

- Real-time statistics (total hosts, active hosts, certificates, request count)
- Recent requests table with auto-refresh
- Quick start guide

**Virtual Hosts**

- Add, edit, delete virtual hosts
- Enable/disable toggle
- Support for wildcard patterns (*.example.com)
- Form validation

**Certificates**

- View all generated certificates
- Download Root CA button
- Installation instructions for macOS/Linux/Windows
- Expiration indicators

**Request Logs**

- Paginated request history
- Auto-refresh toggle (updates every 3 seconds)
- View full request/response details
- JSON pretty-printing
- Filter and search (coming soon)

### Building the UI

The UI is built with:

- React 18
- Vite (build tool)
- Axios (HTTP client)
- React Router (navigation)

To build from source:

```bash
cd web/ui
npm install
npm run build
```

The built files are automatically embedded in the Go binary when using `make build-full`.

For development with hot reload:

```bash
cd web/ui
npm run dev
# UI available at http://localhost:3000
# Proxies API requests to http://localhost:9000
```

## API Endpoints

All API endpoints are prefixed with `/api/v1`.

### Virtual Hosts

| Method | Endpoint | Description |
|---|---|---|
| `GET` | `/vhosts` | List all virtual hosts |
| `POST` | `/vhosts` | Create a new virtual host |
| `GET` | `/vhosts/{id}` | Get virtual host details |
| `PUT` | `/vhosts/{id}` | Update virtual host |
| `DELETE` | `/vhosts/{id}` | Delete virtual host |

### Certificates

| Method | Endpoint | Description |
|---|---|---|
| `GET` | `/certificates` | List all generated certificates (no key material) |
| `DELETE` | `/certificates/{id}` | Delete a certificate (soft delete) |
| `GET` | `/certificates/ca` | Download Root CA certificate |
| `POST` | `/certificates/ca/replace` | Replace Root CA with custom subject fields |

### Request Logs

| Method | Endpoint | Description |
|---|---|---|
| `GET` | `/logs?limit=100&offset=0` | List captured requests (paginated) |
| `GET` | `/logs/{id}` | Get request details with full headers/body |

### Health

| Method | Endpoint | Description |
|---|---|---|
| `GET` | `/health` | Health check endpoint |

## Architecture

```
Client -> [DNS Resolution] -> SPECTRE Proxy
         (127.0.0.1)
                         ↓
                    [TLS Handshake]
                 (On-Demand Cert Generation)
                         ↓
                   [Request Interception]
                     (Capture & Log)
                         ↓
                   [Forward to Backend]
               (Establish Real TLS Connection)
                         ↓
                   Backend Server
```

### Class Diagram

```
┌──────────────────┐     1         ┌──────────────────────────┐
│    cmd/spectre   │───uses ──────▶│          Config           │
└──────────────────┘1              └──────────────────────────┘
         │
         │ wires
         ▼
┌────────────────────────────────────────────────────────────────┐
│                            main.go                              │
│  storage.NewDatabase() -> cert.NewManager() -> vhost.Router()  │
  │ -> proxy.NewProxy() -> api.NewServer()                        │
└────────────────────────────────────────────────────────────────┘
         │            │            │            │
         │            │            │            │
         ▼            ▼            ▼            ▼
┌─────────────┐ ┌────────────┐ ┌────────────┐ ┌──────────┐
│   storage   │ │    cert    │ │   vhost    │ │   api    │
└──────┬──────┘ └──────┬─────┘ └──────┬─────┘ └────┬─────┘
       │               │              │             │
       │    ┌──────────┼──────────────┤             │
       │    │     ┌────┴─────────────┐│             │
       │    │     │     api/handlers  ││             │
       │    │     │ (VHost, Cert,   ││             │
       │  ┌─┴────▶│   Request)      ││             │
       │  │      └──────────────────┘│             │
       │  │                           │             │
       ▼  ▼                           ▼             │
┌──────────────────┐      ┌──────────────────────┐  │
│  Proxy           │      │  Server              │  │
│  - httpListener  │      │  - certManager       │  │
│  - httpsListener │      │  - router            │  │
│  - certManager───┼─────▶│  - db                │  │
│  - router        │      │  - handlers          │  │
│  - requestRepo   │      └──────────────────────┘  │
└──────────────────┘                                ▲
                                                    │
                                              ┌─────┴──────────────────┐
                                              │    internal/web         │
                                              │  (embedded static files) │
                                              └──────────────────────────┘

┌─────────────────────────────────────────────────────────────────────┐
│                              storage                                 │
│  ┌──────────────┐  ┌─────────────────────┐  ┌──────────────────┐   │
│  │ Database     │→ │ CertRepository      │→ │ VHostRepository  │   │
│  │ *sql.DB      │  │ - SaveRootCA       │   │ - Create         │   │
│  │ migrate()    │  │ - GetRootCA        │   │ - GetByHostname  │   │
│  │ Tables:      │  │ - SaveCertificate  │   │ - List           │   │
│  │  root_ca     │  │ - ListCertificates │   │ - Update         │   │
│  │  virtual_hosts │-│ - DeleteCertificate │  │ - Delete         │   │
│  │  certificates│  └─────────────────────┘   └──────────────────┘   │
│  │  request_logs│                    ┌─────────────────────┐        │
│  └──────┬───────┘                    │ RequestRepository │        │
│         │                             │ - Save            │        │
│         ├─────────────────────────────┤ - GetByID         │        │
│         │                             │ - List            │        │
│         │                             │ - DeleteOlderThan │        │
│         │                             │ - Count           │        │
│         │                             └─────────────────────┘        │
│         ▼                                                              │
│  ┌────────────────────────────────────────────────────────────────┐   │
│  │ Models: RootCA | VirtualHost | Certificate | RequestLog       │   │
│  └────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────┐
│                                 cert                                │
│  ┌──────────────┐  ┌──────────────┐  ┌────────────────────────┐    │
│  │ CA           │  │ Generator    │  │ Manager                │    │
│  │ NewCA()      │→ │ Generate()   │  │ GetCertificate()       │    │
│  │ LoadCA()     │  └──────────────┘  │ GetRootCACert()        │    │
│  │ GetCertPEM() │                     │ ReplaceCA()            │    │
│  │ GetKeyPEM()  │                    │ CleanupExpiredCache()  │    │
│  └──────────────┘                     └────────────────────────┘    │
│                                      ┌──────────────┐               │
│                                      │ Cache        │               │
│                                      │ Get/Set      │               │
│                                      │ Delete/Clear │               │
│                                      │ CleanupExpired│               │
│                                      └──────────────┘               │
└─────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────┐
│                               vhost                                  │
│  ┌──────────────────────────────────────────────────────────┐       │
│  │ Router                                                    │       │
│  │ - Add() -exact & wildcard matching                       │       │
│  │ - Match() - hostname lookup (exact then wildcard)        │       │
│  │ - Remove() - deletion                                    │       │
│  │ - LoadFromRepository() - bulk load from DB               │       │
│  └──────────────────────────────────────────────────────────┘       │
└─────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────┐
│                               proxy                                 │
│  ┌────────────────────────────────────────────────────────────┐     │
│  │ Proxy                                                      │     │
│  │ - Start()    - listen on HTTP (8080) + HTTPS (8443)        │     │
│  │ - Shutdown() - graceful shutdown                           │     │
│  │ - handleHTTPConnection()   - plain HTTP handling           │     │
│  │ - handleHTTPSConnection()  - TLS handling + SNI extraction │     │
│  │ - forwardHTTPRequest()     - forward + capture + log       │     │
│  │ - forwardHTTPSRequest()    - forward backend TLS + log     │     │
│  └────────────────────────────────────────────────────────────┘     │
│  - RequestCapture │ ResponseCapture (snapshot structs)               │
└─────────────────────────────────────────────────────────────────────┘
```

### Key Design Patterns

- **Repository Pattern**: Three concrete repositories (`CertRepository`, `VHostRepository`, `RequestRepository`) wrap SQLite persistence
- **Shared In-Memory Router**: The `*vhost.Router` is shared between `proxy` (runtime lookups) and `api/handlers` (CRUD operations), keeping routing rules in sync
- **Lazy Certificate Generation**: `cert.Manager.GetCertificate` implements the `tls.Config.GetCertificate` hook — certificates are generated on first demand for each SNI hostname, cached in-memory with TTL, and persisted to SQLite
- **Two-Server Architecture**: Two independent servers run concurrently:
  - **Proxy** (ports 8080/8443) — raw TCP/TLS listeners that forward traffic
  - **API** (port 9000) — chi-based REST API with embedded static frontend
- **Soft Deletes**: `Certificate` and `RootCA` use an `IsActive` boolean flag rather than physical deletion

## Project Structure

```
spectre/
├── cmd/spectre/          # Main entry point (wires all components)
├── internal/
│   ├── api/              # REST API server (chi router)
│   │   └── handlers/     # HTTP handlers for vhosts, certificates, logs
│   ├── cert/             # Certificate management (CA generation, on-demand certs, cache)
│   ├── config/           # Configuration loading (viper + YAML)
│   ├── proxy/            # Core proxy (HTTP/HTTPS handlers, request capture)
│   ├── storage/          # SQLite data layer (models + repositories)
│   ├── vhost/            # Virtual host routing (exact + wildcard)
│   ├── web/              # Embedded static files (React build artifacts)
│   └── websocket/        # Reserved for future features
├── web/ui/               # React web UI (optional, built with Vite)
├── containers/           # Containerfile for container builds
├── spectre.yaml.example  # Configuration reference
└── Makefile
```

## Development

```bash
# Run tests
make test

# Run with coverage
make test-coverage

# Format code
make fmt

# Run linter
make lint

# Clean build artifacts
make clean
```

## Security Considerations

- **Private Key Protection**: The Root CA private key is stored in the SQLite database. Protect this file.
- **Trust Store**: Only install the Root CA in systems you control. Anyone with the CA can create trusted certificates.
- **HTTPS to Backend**: The proxy validates real backend certificates by default. Do not disable verification in production.
- **API Access**: The management API defaults to localhost binding. Use authentication if exposing externally.

## License

see LICENSE file for details

## Disclaimer

This tool is intended for authorized security testing, reverse engineering, and educational purposes only. Users are responsible for complying with all applicable laws and obtaining proper authorization before intercepting network traffic.

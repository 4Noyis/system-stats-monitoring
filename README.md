# System Stats Monitoring
System Stats Monitoring is a client-server application designed to collect, store, and visualize system performance metrics from multiple hosts. It provides a centralized dashboard for administrators to monitor the health and activity of machines on their network in near real-time.

## Table of Contents

- [Project Purpose](#project-purpose)
- [Features](#features)
- [Tech Stack](#tech-stack)
- [Folder & File Structure](#folder--file-structure)
- [How It Works](#how-it-works)
  - [Client Agent](#client-agent)
  - [Server Application](#server-application)
- [Getting Started (Running Locally)](#getting-started-running-locally)
  - [Prerequisites](#prerequisites)
  - [1. Setup InfluxDB (Database)](#1-setup-influxdb-database)
  - [2. Configure and Run the Server](#2-configure-and-run-the-server)
  - [3. Configure and Run the Client Agent](#3-configure-and-run-the-client-agent)
- [API Endpoints](#api-endpoints)
  - [Client to Server](#client-to-server)
  - [Admin Panel to Server](#admin-panel-to-server)


## Project Purpose
The goal of this project is to provide a robust solution for:
- **Collecting** key system metrics (CPU, memory, disk, network, processes) from various hosts.
- **Storing** this time-series data efficiently in a central database.
- **Visualizing** these metrics through a user-friendly web interface, allowing for monitoring and analysis of host performance.

## Features
**Client Agent:**
- Collects system info, CPU usage, memory usage, disk usage (for `/`), network traffic, and high-resource processes.
- Sends data periodically to the central server in JSON format.

**Server:**
- Receives data from multiple clients via a REST API (built with Gin).
- Stores metrics in InfluxDB v2.x.
- Provides API endpoints for the admin panel to query stored metrics.
- Includes CORS support for frontend interaction.

**Admin Web Panel (Conceptual - to be fully implemented by the user):**
- Displays an overview of all monitored hosts and their current status.
- Provides detailed views for individual hosts.

## Tech Stack
- **Backend (Server & Client Agent):** Go (Golang)
  - `gopsutil`: For system metric collection (client).
  - `gin-gonic/gin`: Web framework for the server API.
  - `influxdb-client-go/v2`: InfluxDB Go client library.
  - `gin-contrib/cors`: CORS middleware for Gin.
- **Database:** InfluxDB v2.x (Time-series database)
- **Containerization:** Docker (for InfluxDB)

## Folder & File Structure
```
system-stats-monitoring/
├── cmd/ # Main application entrypoints
│ ├── monitor/ # Client agent application
│ │ └── main.go
│ └── server/ # Server application
│ └── main.go
├── internal/ # Application-specific internal logic
│ ├── logger/ # Custom logging package (shared)
│ │ └── logger.go
│ ├── stats/ # Client: System stats collection logic
│ │ └── stats.go
│ └── server/ # Server: Internal logic
│ ├── api/ # API handlers (stats_handler.go, dashboard_handler.go)
│ ├── config/ # Server configuration (config.go for InfluxDB, etc.)
│ ├── database/ # Database interaction (influxdb_writer.go, influxdb_reader.go)
│ └── models/ # Server-side data models (payload.go, dashboard_models.go)
├── pkg/ # Exportable packages (e.g., for client data sending)
│ └── exporter/ # Client: JSON exporter and HTTP sender
│ └── exporter.go
├── go.mod
├── go.sum
└── README.md
```

## How It Works

### Client Agent
1.  **Collects Metrics:** Runs on each monitored host. Every 5 seconds (configurable), it gathers:
    - System Info (Hostname, HostID, OS, Kernel, Uptime)
    - CPU (Model, Cores, Usage %)
    - Memory (Total, Used, Usage %)
    - Network (Upload/Download speed)
    - Disk Usage (for `/` path: Total, Used, Usage %)
    - Processes (PID, Name, CPU %, Mem %) exceeding a defined threshold (e.g., >10% CPU or RAM).
2.  **Formats Data:** Aggregates collected metrics into a single Go struct.
3.  **Sends Data:** Serializes the struct to JSON and sends it via an HTTP POST request to the server's `/api/stats` endpoint.

### Server Application
1.  **Receives Data:** The Gin server listens for incoming POST requests on `/api/stats`.
2.  **Validates & Parses:** It validates the request and parses the JSON payload into a Go struct.
3.  **Stores Data in InfluxDB:**
    - Connects to an InfluxDB instance.
    - Transforms the received data into InfluxDB "points."
    - Each point includes:
        - A **measurement** name (e.g., `system_metrics`, `disk_metrics`, `process_metrics`).
        - **Tags** for indexing (e.g., `host_id`, `hostname`, `path` for disk, `pid` for process).
        - **Fields** holding the actual metric values (e.g., `cpu_usage_percent`, `mem_total_gb`).
        - A **timestamp** (from when the client collected the data).
    - Writes these points to the configured InfluxDB bucket.
4.  **Serves Admin Panel API:** Exposes GET endpoints (e.g., `/api/dashboard/...`) that the admin web panel uses to query data from InfluxDB. These endpoints use Flux queries to retrieve and aggregate data.

## Getting Started (Running Locally)

### Prerequisites
- Go (version 1.18+ recommended)
- Docker and Docker Compose (or just Docker if you prefer to manage InfluxDB manually)
- Node.js and npm/yarn (for the admin web panel)
- Git

### 1. Setup InfluxDB (Database)
You need an InfluxDB v2.x instance running.

**Using Docker:**
```bash
# Create a directory for persistent InfluxDB data
mkdir influxdb_data

# Run InfluxDB container (replace placeholders with your desired credentials)
docker run --name influxdb-sysmon -p 8086:8086 -d \
  -e DOCKER_INFLUXDB_INIT_MODE=setup \
  -e DOCKER_INFLUXDB_INIT_USERNAME=my-admin \
  -e DOCKER_INFLUXDB_INIT_PASSWORD=my-super-secret-password \
  -e DOCKER_INFLUXDB_INIT_ORG=my-org \
  -e DOCKER_INFLUXDB_INIT_BUCKET=system_stats \
  -v "$(pwd)/influxdb_data":/var/lib/influxdb2 \
  influxdb:2.7 # Or latest 2.x version
```
 - Access the InfluxDB UI at http://localhost:8086.
 - During the initial setup (if you didn't use DOCKER_INFLUXDB_INIT_ vars, or to get a token):
    - Create an organization (e.g., my-org).
    - Create a bucket (e.g., system_stats).
    - Generate an API Token with write access to your bucket and read access. Note this token down. (Navigate to "Load Data" -> "API Tokens" -> "Generate API Token" -> "All Access Token" or a custom one with appropriate permissions).

### 2. Configure and Run the Server
 - Clone the repository (if you haven't):
 ```bash
git clone https://github.com/4Noyis/system-stats-monitoring.git
cd system-stats-monitoring
```

### 3. Set Environment Variables for the Server:
Open a terminal where you will run the server.
```bash
export INFLUXDB_URL="http://localhost:8086"
export INFLUXDB_TOKEN="YOUR_INFLUXDB_API_TOKEN" # Paste the token you generated
export INFLUXDB_ORG="my-org"                     # Your InfluxDB organization
export INFLUXDB_BUCKET="system_stats"            # Your InfluxDB bucket
```

### 4. Run the Server
```bash
go run cmd/server/main.go
```
The server should start, connect to InfluxDB, and listen on port 8080

## 3. Configure and Run the Client Agent
1. Open a new terminal
2. Navigate to the client agent's directory
3. Run the Client Agent:
```bash
go run cmd/monitor/main.go
```
The client will start collecting metrics and sending them to http://localhost:8080/api/stats. Check the server logs to see incoming data and InfluxDB write confirmations.
You can run multiple instances of the client on different machines (or simulate by running it multiple times locally if it generates unique HostIDs, though true uniqueness comes from different machines).


## API Endpoint 
### Client to server

- POST /api/stats:
    - Purpose: Client agents send their collected metrics to this endpoint.
    - Request Body: JSON object containing AllHostStats (system, CPU, memory, disk, network, processes).
    - Headers: Content-Type: application/json.
    - Response: 200 OK on success, error codes on failure.

- Admin Panel to Server
    -GET /api/dashboard/hosts/overview:
    Purpose: Get a summary list of all monitored hosts and their latest key metrics.
    Response: JSON array of HostOverviewData.
    - GET /api/dashboard/host/:hostID/details:
    Purpose: Get detailed metrics, OS/hardware info, and recent process list for a specific host.
    URL Parameter: :hostID - The unique ID of the host.
    Response: JSON object of HostDetailsData.
    - GET /api/dashboard/host/:hostID/metrics/:metricName:
    Purpose: Get historical time-series data for a specific metric of a host (for charts).
    - URL Parameters:
        - :hostID - The unique ID of the host.
        - :metricName - The name of the field to query (e.g., cpu_usage_percent, mem_usage_percent).
    Query Parameters (Optional):
        - range (e.g., 1h, 30m): Time duration to look back.
        - aggregate (e.g., 30s, 1m): Aggregation window for time-series data.
        - Response: JSON array of MetricPoint objects ({timestamp: "HH:MM", value: number}).
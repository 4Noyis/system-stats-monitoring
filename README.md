# system-stats-monitoring

system-stats-monitoring/
├── cmd/                      # Main application entrypoints
│   └── monitor/              # You can have multiple commands
│       └── main.go           # Entry point
├── internal/                 # Application-specific internal logic
│   ├── stats/                # e.g. CPU, memory, disk stats logic
│   │   └── stats.go
│   └── logger/               # Custom logging package
│       └── logger.go
├── pkg/                      # Exportable packages for other apps
│   └── exporter/             # JSON/XML exporters
│       └── exporter.go
├── go.mod
└── README.md

This structure is for my system monitoring project. Its basically gets:
- System info: Host ID & Host name, Operating system, Kernel, Uptime.
- CPU: Model, Cores, Usage
- Memory: Total, Usage
- Network: Download, Upload
- Working processes
For every host(PC, server etc.) on the same network it gets their CPU Usage, Memory Usage, Working processes, and network usage -download and upload- every 5 second and send them as a JSON format to central server with using HTTP POST request. On this server data will be stored with using InfluxDB (WebSocket for real time updates).  
# system-stats-monitoring

```
system-stats-monitoring/
├── cmd/
│   ├── monitor/              # Client
│   │   └── main.go
│   └── server/               # Server application
│       └── main.go           # Server entry point
├── internal/
│   ├── stats/                # Client stats collection logic
│   ├── logger/               # Shared logger 
│   └── server/               # Server-specific internal logic
│       ├── api/              # API handlers 
│       ├── config/           # Server configuration 
│       ├── database/         # Database interaction logic 
│       └── models/           # Data models for server-side
├── pkg/
│   └── exporter/             # Client exporter
├── go.mod
├── go.sum
└── README.md
```

This structure is for my system monitoring project. Its basically gets:
- System info: Host ID & Host name, Operating system, Kernel, Uptime.
- CPU: Model, Cores, Usage
- Memory: Total, Usage
- Network: Download, Upload
- Working processes
For every host(PC, server etc.) on the same network it gets their CPU Usage, Memory Usage, Working processes, and network usage -download and upload- every 5 second and send them as a JSON format to central server with using HTTP POST request. On this server data will be stored with using InfluxDB (WebSocket for real time updates).  
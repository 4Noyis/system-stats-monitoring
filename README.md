# system-stats-monitoring

```
system-stats-monitoring/
├── cmd
│   ├── monitor
│   │   └── main.go
│   └── server
│       └── main.go
├── internal
│   ├── logger
│   │   └── logger.go
│   ├── server
│   │   ├── api
│   │   │   ├── dashboard_handler.go
│   │   │   └── stats_handler.go
│   │   ├── config
│   │   │   └── config.go
│   │   ├── database
│   │   │   ├── influxdb_reader.go
│   │   │   └── influxdb_writer.go
│   │   └── models
│   │       ├── dashboard_models.go
│   │       └── payload.go
│   └── stats
│       └── stats.go
├── pkg
│   └── exporter
│       └── exporter.go
├── cpu_log.json
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
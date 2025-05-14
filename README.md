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
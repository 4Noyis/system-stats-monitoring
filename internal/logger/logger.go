package logger

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"os"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
)

type CPUStat struct {
	Timestamp string  `json:"timestamp" xml:"timestamp"`
	Usage     float64 `json:"usage" xml:"usage"`
}

func LogCPUStats(interval time.Duration, count int, format string, filename string) error {
	var logs []CPUStat

	for i := 0; i < count; i++ {
		percent, err := cpu.Percent(time.Second, false)
		if err != nil {
			return err
		}

		log := CPUStat{
			Timestamp: time.Now().Format(time.RFC3339),
			Usage:     percent[0],
		}
		logs = append(logs, log)

		fmt.Printf("[%s] CPU: %.2f%%\n", log.Timestamp, log.Usage)
		time.Sleep(interval)
	}

	var data []byte
	var err error

	switch format {
	case "json":
		data, err = json.MarshalIndent(logs, "", "  ")
	case "xml":
		data, err = xml.MarshalIndent(logs, "", "  ")
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}

	if err != nil {
		return err
	}

	return os.WriteFile(filename, data, 0644)
}

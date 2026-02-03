# gniot

A lightweight IoT device management server written in Go. GNIOT allows you to register IoT devices, define actions, and schedule jobs that execute actions on devices via JSON-RPC.

## Features

- **Device Management** - Register and manage IoT devices with their network configuration
- **Action Definitions** - Define reusable actions with JSON-RPC method paths and parameters
- **Job Scheduling** - Schedule actions to run on devices at specific times with optional intervals
- **Immediate Execution** - Execute actions on devices on-demand via the `/execute` endpoint

## Running the Server

```bash
go run cmd/main.go
```

The server starts on `http://127.0.0.1:8080` by default.

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | Server port |
| `JOBS_INTERVAL` | `1m` | How often the job runner checks for pending jobs |

## API Reference

### Devices

**Create a device**
```bash
curl -X POST http://127.0.0.1:8080/devices \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Living Room Light",
    "type": "light",
    "chip": "ESP32",
    "board": "ESP32-DevKitC",
    "ip": "192.168.1.100",
    "actions": "[1, 2]"
  }'
```

**List all devices**
```bash
curl http://127.0.0.1:8080/devices
```

**Get a device by ID**
```bash
curl http://127.0.0.1:8080/devices/1
```

**Update a device**
```bash
curl -X POST http://127.0.0.1:8080/devices/1 \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Kitchen Light",
    "ip": "192.168.1.101"
  }'
```

**Delete a device**
```bash
curl -X DELETE http://127.0.0.1:8080/devices/1
```

### Actions

**Create an action**
```bash
curl -X POST http://127.0.0.1:8080/actions \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Turn On",
    "path": "light.set",
    "params": "{\"state\": \"on\"}"
  }'
```

**List all actions**
```bash
curl http://127.0.0.1:8080/actions
```

**Get an action by ID**
```bash
curl http://127.0.0.1:8080/actions/1
```

**Update an action**
```bash
curl -X POST http://127.0.0.1:8080/actions/1 \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Turn Off",
    "params": "{\"state\": \"off\"}"
  }'
```

**Delete an action**
```bash
curl -X DELETE http://127.0.0.1:8080/actions/1
```

### Jobs

**Create a job**
```bash
curl -X POST http://127.0.0.1:8080/jobs \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Morning Lights",
    "devices": "[1, 2]",
    "action": "1",
    "run_at": "2025-02-03T07:00:00Z",
    "interval": "24h"
  }'
```

**List all jobs**
```bash
curl http://127.0.0.1:8080/jobs
```

**Get a job by ID**
```bash
curl http://127.0.0.1:8080/jobs/1
```

**Update a job**
```bash
curl -X POST http://127.0.0.1:8080/jobs/1 \
  -H "Content-Type: application/json" \
  -d '{
    "interval": "12h"
  }'
```

**Delete a job**
```bash
curl -X DELETE http://127.0.0.1:8080/jobs/1
```

### Execute

**Execute an action on a device immediately**
```bash
curl -X POST http://127.0.0.1:8080/execute \
  -H "Content-Type: application/json" \
  -d '{
    "deviceId": 1,
    "actionId": 1
  }'
```

## Data Models

### Device
| Field | Type | Description |
|-------|------|-------------|
| `id` | int | Auto-generated ID |
| `name` | string | Device name |
| `type` | string | Device type (e.g., "light", "sensor") |
| `chip` | string | Chip model (e.g., "ESP32") |
| `board` | string | Board model |
| `ip` | string | Device IP address (must be private) |
| `actions` | string | JSON array of action IDs |

### Action
| Field | Type | Description |
|-------|------|-------------|
| `id` | int | Auto-generated ID |
| `name` | string | Action name |
| `path` | string | JSON-RPC method path |
| `params` | string | JSON-encoded parameters |

### Job
| Field | Type | Description |
|-------|------|-------------|
| `id` | int | Auto-generated ID |
| `name` | string | Job name |
| `devices` | string | JSON array of device IDs |
| `action` | string | Action ID |
| `run_at` | string | Next execution time (RFC3339) |
| `interval` | string | Repeat interval (e.g., "1h", "24h", "30m") |

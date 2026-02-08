# gniot

A lightweight IoT device management server written in Go. GNIOT allows you to register IoT devices, define actions, and execute actions on devices via JSON-RPC.

## Features

- **Device Management** - Register and manage IoT devices with their network configuration
- **Action Definitions** - Define reusable actions with JSON-RPC method paths and parameters
- **Immediate Execution** - Execute actions on devices on-demand via the `/execute` endpoint
- **Automations** - Define scheduled automations with triggers, conditions, and actions using YAML definitions

## Running the Server

```bash
go run cmd/main.go
```

The server starts on `http://127.0.0.1:8080` by default.

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | Server port |
| `AUTOMATIONS_INTERVAL` | `1m` | How often the automation runner checks for due automations |

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

### Automations

**Create an automation**
```bash
curl -X POST http://127.0.0.1:8080/automations \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Cool down room",
    "enabled": true,
    "definition": "interval: \"5m\"\ncondition_logic: \"and\"\ntriggers:\n  - device: \"temp_sensor\"\n    action: \"read_temp\"\n    conditions:\n      - field: \"temperature\"\n        operator: \">\"\n        threshold: 25.0\nactions:\n  - device: \"fan\"\n    action: \"turn_on\""
  }'
```

**List all automations**
```bash
curl http://127.0.0.1:8080/automations
```

**Get an automation by ID**
```bash
curl http://127.0.0.1:8080/automations/1
```

**Update an automation**
```bash
curl -X POST http://127.0.0.1:8080/automations/1 \
  -H "Content-Type: application/json" \
  -d '{
    "enabled": false
  }'
```

**Delete an automation**
```bash
curl -X DELETE http://127.0.0.1:8080/automations/1
```

#### Automation Definition (YAML)

The `definition` field is a YAML string that describes the automation logic:

```yaml
interval: "5m"            # How often triggers are evaluated (min: 1s)
condition_logic: "and"    # "and" (default) or "or" — how trigger results combine
triggers:
  - device: "temp_sensor"
    action: "read_temp"
    conditions:
      - field: "temperature"       # Supports nested fields like "sensor.value"
        operator: ">"              # >, <, >=, <=, ==, !=
        threshold: 25.0
actions:
  - device: "fan"
    action: "turn_on"
```

Triggers are evaluated at the specified interval. When conditions are met (combined with the chosen logic), the listed actions are executed on their respective devices.

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

### Automation
| Field | Type | Description |
|-------|------|-------------|
| `id` | int | Auto-generated ID |
| `name` | string | Unique automation name |
| `enabled` | bool | Whether the automation is active |
| `definition` | string | YAML automation definition (triggers, conditions, actions) |
| `lastCheck` | string | RFC3339 timestamp of last check |
| `lastTriggersRun` | string | RFC3339 timestamp of last trigger evaluation |
| `lastActionRun` | string | RFC3339 timestamp of last action execution |

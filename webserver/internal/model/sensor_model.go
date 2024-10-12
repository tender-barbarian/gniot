package model

type Sensor struct {
	ID              int32   `json:"id"`
	Name            string  `json:"name"`
	SensorType      string  `json:"sensorType"`
	Chip            string  `json:"chip"`
	Board           string  `json:"board"`
	IP              string  `json:"ip"`
	SensorMethodIDs []int32 `json:"sensorMethodIDs,omitempty"`
}

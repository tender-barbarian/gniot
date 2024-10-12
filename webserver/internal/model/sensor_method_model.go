package model

type SensorMethod struct {
	ID          int32  `json:"id"`
	Name        string `json:"name"`
	HttpMethod  string `json:"httpMehtod"`
	RequestBody string `json:"requestBody"`
}

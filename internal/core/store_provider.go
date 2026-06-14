package core

type PointValue struct {
	SensorID uint    `json:"sensorId"`
	Point    string  `json:"point"`
	Time     int64   `json:"time"`
	Value    float64 `json:"value"`
}

type StoreProvider interface {
	Write(tsEntities []any)
	QueryMetrics(sensorID uint, point string, start, end int64, step, limit uint) []*PointValue
	QueryAlarms(start, end int64, sensorID, defineID, ruleID uint, onlyActive bool) []any
	Dispose()
}

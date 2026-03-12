package cache

import "time"

const (
	LatestMetricKeyPrefix = "device_metric:"
	LatestMetricKeySuffix = ":latest"
	LatestMetricTTL       = 60 * time.Second
)

func LatestMetricKey(deviceID string) string {
	return LatestMetricKeyPrefix + deviceID + LatestMetricKeySuffix
}

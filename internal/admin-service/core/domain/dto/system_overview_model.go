package dto

type SystemOverview struct {
	Timestamp          string                   `json:"timestamp"`
	Metrics            MetricsParams            `json:"metrics"`
	DriverDistribution DriverDistributionParams `json:"driver_contribution"`
	Hotspots           []HotspotsParams         `json:"hotspots"`
}

type MetricsParams struct {
	ActiveRides                int     `json:"active_rides"`
	AvailableDrivers           int     `json:"available_drivers"`
	BusyDrivers                int     `json:"busy_drivers"`
	TotalRidesToday            int     `json:"total_rides_today"`
	TotalRevenueToday          float32 `json:"total_revenue_today"`
	AverageWaitTimeMinutes     float32 `json:"average_wait_time_minutes"`
	AverageRideDurationMinutes float32 `json:"average_ride_duration_minutes"`
	CancellationRate           float32 `json:"cancellation_rate"`
}
type DriverDistributionParams struct {
	Economy string `json:"ECONOMY"`
	Premium string `json:"PREMIUM"`
	XL      string `json:"XL"`
}

type HotspotsParams struct {
	Location       string `json:"location"`
	ActiveRides    int    `json:"active_rides"`
	WaitingDrivers int    `json:"waiting_drivers"`
}

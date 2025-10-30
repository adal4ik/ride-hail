package messagebrokerdto


//Location Update ← location_fanout exchange
type LocationUpdate struct {
	DriverID       string    `json:"driver_id"`
	RideID         string    `json:"ride_id"`
	Location       Location  `json:"location"`
	SpeedKmh       float64   `json:"speed_kmh"`
	HeadingDegrees float64   `json:"heading_degrees"`
	Timestamp      string `json:"timestamp"`
}

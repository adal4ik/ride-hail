package dto

type RidesCancelRequestDto struct {
	Reason string `json:"reason"`
}

type RideCancelResponseDto struct {
	RideId      string `json:"ride_id"`
	Status      string `json:"status"`
	CancelledAt string `json:"cancelled_at"`
	Message     string `json:"message"`
}

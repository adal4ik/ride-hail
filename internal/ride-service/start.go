package rideservice

type RideService struct{}

func New(port *int) *RideService {
	return &RideService{}
}

func (rs *RideService) Run() {
}

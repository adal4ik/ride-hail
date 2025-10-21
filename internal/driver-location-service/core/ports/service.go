package ports

type IDriverService interface {
	GoOnline()
	GoOffline()
	UpdateLocation()
	StartRide()
	CompleteRide()
}

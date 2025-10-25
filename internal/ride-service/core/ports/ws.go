package ports

import websocketdto "ride-hail/internal/ride-service/core/domain/websocket_dto"

type INotifyWebsocket interface {
	WriteToUser(passengerId string, msg websocketdto.Event)
}

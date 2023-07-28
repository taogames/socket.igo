module socket.igo/v4

go 1.18

replace engine.igo/v4 v4.0.0 => ../engine.igo

require (
	engine.igo/v4 v4.0.0
	go.uber.org/zap v1.24.0
)

require (
	github.com/google/uuid v1.3.0 // indirect
	github.com/gorilla/websocket v1.5.0 // indirect
	github.com/sony/sonyflake v1.1.0 // indirect
	go.uber.org/atomic v1.7.0 // indirect
	go.uber.org/multierr v1.6.0 // indirect
)

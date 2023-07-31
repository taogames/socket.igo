package main

import (
	"fmt"
	"net/http"
	"time"

	socketigo "github.com/taogames/socket.igo"
	"go.uber.org/zap"
)

// Test cases from https://github.com/socketio/socket.io-protocol

func main() {
	fmt.Println("Testing test suite")

	// logger, err := zap.NewDevelopment()
	// if err != nil {
	// 	panic(err)
	// }

	conf := zap.Config{
		Level:            zap.NewAtomicLevelAt(zap.InfoLevel),
		Development:      true,
		Encoding:         "console",
		EncoderConfig:    zap.NewDevelopmentEncoderConfig(),
		OutputPaths:      []string{"stderr"},
		ErrorOutputPaths: []string{"stderr"},
	}
	logger, err := conf.Build()
	if err != nil {
		panic(err)
	}

	server := socketigo.NewServer(
		socketigo.WithPingInterval(time.Millisecond*300),
		socketigo.WithPingTimeout(time.Millisecond*200),
		socketigo.WithMaxPayload(1000000),
		socketigo.WithLogger(logger.Sugar()),
	)

	server.Of("/").OnConnection(func(socket *socketigo.Socket) {

		time.Sleep(time.Millisecond * 100)

		socket.Emit("auth", socket.Handshake.Auth)

		socket.On("message", func(args ...interface{}) {
			socket.Emit("message-back", args...)
		})
	})

	server.Of("/custom").OnConnection(func(socket *socketigo.Socket) {

		time.Sleep(time.Millisecond * 100)

		socket.Emit("auth", socket.Handshake.Auth)

		socket.On("message", func(args ...interface{}) {
			socket.Emit("message-back", args...)
		})
	})

	go server.Accept()

	http.Handle("/socket.io/", server)
	if err := http.ListenAndServe(":3000", nil); err != nil {
		panic(err)
	}
}

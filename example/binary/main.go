package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	socketigo "github.com/taogames/socket.igo"
	"go.uber.org/zap"
)

func main() {
	fmt.Println("Testing binary")

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
		socketigo.WithMaxPayload(10000),
		socketigo.WithLogger(logger.Sugar()),
	)

	server.Of("/").OnConnection(func(socket *socketigo.Socket) {
		socket.On("upload", func(data []byte) {
			if err := os.WriteFile("./file", data, 0755); err != nil {
				fmt.Println(err)
			}
		})
	})

	go server.Accept()

	http.Handle("/socket.io/", server)
	if err := http.ListenAndServe(":3000", nil); err != nil {
		panic(err)
	}
}

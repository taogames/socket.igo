# socket.igo
\[[English](README.md)\]
\[[简体中文](README-zh-Hans.md)\]

A go implementation of [Socket.IO](https://socket.io/) server 4.x.

socket.igo is based on [engine.igo](https://github.com/taogames/engine.igo)。


## Compatibility
Socket.IO client ver >= 3.0.0


## Installation
```
go get github.com/taogames/socket.igo
```


## Example
* [Chat Room](example/chat)


## Get Started
```go
	server := socketigo.NewServer()

	server.Of("/").OnConnection(func(socket *socketigo.Socket) {
        fmt.Println("Connected")

		socket.On("hello", func(msg string) {
			socket.Emit("world", msg)
		})

        socket.OnDisconnect(func(reason socketigo.DisconnectReason) {
            fmt.Println("Disconnected")
		})
	})

	go server.Accept()
	http.Handle("/socket.io/", server)
	if err := http.ListenAndServe(":3000", nil); err != nil {
		panic(err)
	}
```


## Contributing
We welcome your opinions, discussions and contributions to this project. There are quite a few to-dos including but not limited to:
* Feature: Dynamic namespace
* Test: Unit test, Performance test
* Client
* ......

Feel free to contact us at telegram group: [socket.igo](https://t.me/+9c2-MZrtT4tmMTJl)


## License
[Apache License 2.0](LICENSE)

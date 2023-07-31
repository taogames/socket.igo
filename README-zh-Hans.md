# socket.igo
\[[English](README.md)\]
\[[简体中文](README-zh-Hans.md)\]

Golang 实现的 [Socket.IO](https://socket.io/) 4.x 版服务端.

socket.igo 底层基于 [engine.igo](https://github.com/taogames/engine.igo)。


## 兼容性
Socket.IO 客户端版本 >= 3.0.0


## 安装
```
go get github.com/taogames/socket.igo
```


## 示例
* [Chat Room](example/chat)


## 开始
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


## 贡献
欢迎大伙一起来讨论&贡献代码，一起提高项目质量。包括不限于：
* 功能方面：动态域名, 二进制消息, 消息确认
* 测试方面：单元测试，性能测试
* 客户端
* ......


欢迎加入电报群: [socket.igo](https://t.me/+9c2-MZrtT4tmMTJl)
## 许可证
[Apache License 2.0](LICENSE)

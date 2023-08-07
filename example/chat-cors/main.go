package main

import (
	"net/http"

	"github.com/gorilla/handlers"
	socketigo "github.com/taogames/socket.igo"
)

var numUsers int

func main() {
	server := socketigo.NewServer()
	server.Of("/").OnConnection(func(socket *socketigo.Socket) {
		addedUser := false
		socket.On("add user", func(username string) {
			if addedUser {
				return
			}

			socket.Custom["username"] = username
			numUsers++
			addedUser = true

			socket.Emit("login", struct {
				NumUsers int `json:"numUsers"`
			}{
				NumUsers: numUsers,
			})

			socket.Broadcast().Emit("user joined", struct {
				Username string `json:"username"`
				NumUsers int    `json:"numUsers"`
			}{
				Username: socket.Custom["username"].(string),
				NumUsers: numUsers,
			})
		})

		socket.On("new message", func(data string) {
			socket.Broadcast().Emit("new message", struct {
				Username string `json:"username"`
				Message  string `json:"message"`
			}{
				Username: socket.Custom["username"].(string),
				Message:  data,
			})
		})

		socket.On("typing", func() {
			socket.Broadcast().Emit("typing", struct {
				Username string `json:"username"`
			}{
				Username: socket.Custom["username"].(string),
			})
		})

		socket.On("stop typing", func() {
			socket.Broadcast().Emit("stop typing", struct {
				Username string `json:"username"`
			}{
				Username: socket.Custom["username"].(string),
			})
		})

		socket.OnDisconnect(func(reason socketigo.DisconnectReason) {

			if addedUser {
				numUsers--
			}
			socket.Broadcast().Emit("user left", struct {
				Username string `json:"username"`
				NumUsers int    `json:"numUsers"`
			}{
				Username: socket.Custom["username"].(string),
				NumUsers: numUsers,
			})
		})
	})

	go server.Accept()

	router := http.NewServeMux()
	router.Handle("/socket.io/", server)
	router.Handle("/", http.FileServer(http.Dir("")))

	if err := http.ListenAndServe(":3000", handlers.CORS()(router)); err != nil {
		panic(err)
	}
}

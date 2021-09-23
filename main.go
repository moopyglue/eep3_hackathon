package main

import (
	"fmt"
	"net/http"

	"github.com/gorilla/websocket"
)

var controller_display_channel = make(chan string, 1000)
var display_controller_channel = make(chan string, 1000)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func controllerReader(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("Controller reading from: %q\n", r.URL.Path)
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println(err)
		return
	}
	_readMessage(*conn, controller_display_channel)
}

func displayWriter(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("Display writing to: %q\n", r.URL.Path)
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println(err)
		return
	}
	_writeMessage(*conn, controller_display_channel)
}

func displayReader(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("Display reading from: %q\n", r.URL.Path)
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println(err)
		return
	}
	_readMessage(*conn, display_controller_channel)
}

func controllerWriter(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("Controller writing to: %q\n", r.URL.Path)
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println(err)
		return
	}
	_writeMessage(*conn, display_controller_channel)
}

func _writeMessage(conn websocket.Conn, channel chan string) {
	for {
		message := <-channel
		var err = conn.WriteMessage(1, []byte(message))
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Printf("Write: %s\n", message)
	}
}

func _readMessage(conn websocket.Conn, channel chan string) {
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			fmt.Println(err)
			return
		}
		channel <- string(message)
		fmt.Printf("Read: %s\n", message)
	}
}

func main() {
	listenPort := "6001"

	http.HandleFunc("/controller_send", controllerReader)
	http.HandleFunc("/controller_get", displayWriter)
	http.HandleFunc("/display_send", displayReader)
	http.HandleFunc("/display_get", controllerWriter)
	http.Handle("/", http.FileServer(http.Dir("./html/")))

	fmt.Printf("Starting on Port:%s\n", listenPort)
	err := http.ListenAndServe(":"+listenPort, nil)
	if err != nil {
		panic("ListenAndServe: " + err.Error())
	}
}

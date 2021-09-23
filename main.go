package main

import (
	"fmt"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

type msg struct {
	str []byte
}

var controllerMessage atomic.Value
var sleepTimer time.Duration = 9000

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func controllerReader(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("controller reading:: %q\n", r.URL.Path)
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println(err)
		return
	}
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			fmt.Println(err)
			return
		}
		v := msg{message}
		controllerMessage.Store(v)
		fmt.Printf("Message from controller -> %s\n", message)
	}
}

func displayWriter(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("display writing: %q\n", r.URL.Path)
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println(err)
		return
	}
	for {
		v := controllerMessage.Load().(msg)
		var err = conn.WriteMessage(1, v.str)
		if err != nil {
			fmt.Println(err)
			return
		}
		time.Sleep(sleepTimer * time.Millisecond)
		fmt.Printf("To display -> %s\n", v.str)
	}
}

func main() {
	listenPort := "6001"
	v := msg{[]byte("NULL")}
	controllerMessage.Store(v)

	http.HandleFunc("/controller_send", controllerReader)
	http.HandleFunc("/controller_get", displayWriter)
	http.HandleFunc("/display_send", controllerReader)
	http.HandleFunc("/display_get", displayWriter)
	http.Handle("/", http.FileServer(http.Dir("./html/")))

	fmt.Printf("Starting on Port:%s\n", listenPort)
	err := http.ListenAndServe(":"+listenPort, nil)
	if err != nil {
		panic("ListenAndServe: " + err.Error())
	}
}


//
// Haloween eyes in teh skey
// SImon Wall 2020
//

package main

import (
    "bytes"
    "log"
    "sync"
    "os"
	"time"
	"net/http"
	"github.com/gorilla/websocket"
    "github.com/timtadh/getopt"
)

// track latest sender messages
type message struct {
    value  []byte
    mtime  int64
}
var messages = make(map[string]message)

// track links between sender and getter
type link struct {
    value  string
    mtime  int64
}
var links = make(map[string]link)

var runtype = "none"

//simple Read/Write mux 
var mux = &sync.RWMutex{}

// track active getter list
var active_getters = make(map[string]string)

// upgrader used to convert https to wss connections
var upgrader = websocket.Upgrader{
    ReadBufferSize:  10240,
    WriteBufferSize: 10240,
    CheckOrigin: func(r *http.Request) bool {
        return true
    },
}

// New Setter (Controller) connection
// 
// Establish a  session where a setter(controller) passes details of 
// the getter(web page) it wishes to connect to and then passes in messages which
// we then pass on to any listening getters (web pages) that are linked to that controller

func sendSession(w http.ResponseWriter, r *http.Request) {

    // The URL for the connection has 2 query values
    // sid = Sender ID
    // gid = Getter id

    sid:=r.URL.Query().Get("s");
    gid:=r.URL.Query().Get("g");
    logtag:=sid+":  in: ";
    log.Print(logtag,"started: adding in client ",gid)

    // Upgrade to a Web siocket connection
    //
    // a web socket connection starts life asa an http connection
    // a https:// becomes a wss:// and a http:// becomes a ws://
    // this enables the connection to be fully established before 
    // it is repurposed to reduce the complexity of the websocket protocol 
    // and let t focus on the providing a fast responsive connection

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
        log.Print(logtag,err);
        log.Print(logtag,"exiting")
        return
    }

    // define the link between the getter (web page) and controller
    // (mobile phone?), this lets us map a single controller to 
    // multiple web pages at the same time or visa versa

    crossLink(gid,sid)

    // Now we enter an infinate loop where we
    // read messages from the sender(controller) and then 
    // use 'saveMessage()' to pass them to any listening
    // getters(web pages)
    // - a lost connection will end the loop
    // - ReadMessage() is a blocking read so waits without using resources
	for {
		_, rec_mess, err := conn.ReadMessage()
		if err != nil {
            log.Print(logtag,err);
            log.Print(logtag,"exiting")
            sendUnlink(sid)
            return
        }
        saveMessage(sid,rec_mess)
	}
}

// New Getter (Web Page) connection
// 
// Establish a session where a getter(web page) listens for messages.
// It relies on a seperate setter (controller) connection to supply the
// incoming messages, it does not care where they are from.

func getSession(w http.ResponseWriter, r *http.Request) {

    // The URL for the connection has 2 query values
    // id = getter(web page) ID
    // NOTE: it relies on the controller to reach out to it
    id:=r.URL.Query().Get("g");
    logtag:=id+": out: ";
    log.Print(logtag,"started")

    // Upgrade to a Web siocket connection
    //
    // a web socket connection starts life asa an http connection
    // a https:// becomes a wss:// and a http:// becomes a ws://
    // this enables the connection to be fully established before 
    // it is repurposed to reduce the complexity of the websocket protocol 
    // and let t focus on the providing a fast responsive connection

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
        log.Print(logtag,err);
        log.Print(logtag,"exiting")
        return
    }

    // register the getter
    // if this is a reconnection of the se same session the
    // getter will provide the the same ID so is able to reestablish
    // a temporarily lost connection
    // (active_gettersp[] is a global variable - yuk)
    // ( a MUX is used for simple locking )
    mux.Lock()
    active_getters[id]=""
    mux.Unlock()

    // Now we enter an infinate loop where we
    // just sits and wait for messages from controller
    // A lost connection will end the loop
    // getMessage() does not block and if values dioes not change for 50*100 ms
    // then the loop will resend the last message, this keeps the connection open.

    lastv := []byte{}
    nosend := 0
	for {
        get_mess := getMessage(id)
        if( !bytes.Equal(get_mess, lastv) || nosend > 100 ) {
            // new message or 50*100ms timeout 
            // send message to waiting getter(web page)
	        var err = conn.WriteMessage(1,get_mess)
	        if err != nil {
                log.Print(logtag,err);
                log.Print(logtag,"exiting")
                // ( a MUX is used for simple locking )
                mux.Lock()
                delete(active_getters,id)
                mux.Unlock()
                return
            }
	        log.Print(logtag,string(get_mess))
            nosend = 0
        } else {
            // no new message
            nosend++
        }
	    time.Sleep(50*time.Millisecond);
        lastv=get_mess;
	}
}

// save(send) a message against a named setter(controller)
// in the messages[] array with the message and time it was 
// last updated.
// if a new controller message turns up before the reader 
// has seen the change then the getter only sees the latest message

func saveMessage(id string,mess []byte) {

	log.Print("saveMessage: send,"+id+string(mess))

    m := message{value:mess}
    m.mtime=time.Now().Unix()
    // ( a MUX is used for simple locking )
    mux.Lock()
    messages[id]=m
    mux.Unlock()

}

// get(recvieve) a message by checking the corisponding 
// setter(controller)
// NOTE: this eimplemntation means it will not work  for
//     one*getter / many*setters right now.

func getMessage(gid string) (mess []byte){

    // ( a MUX is used for simple locking )
    mux.RLock()
    if sid,ok := links[gid]; ok {
        if m,ok := messages[sid.value]; ok {
            mux.RUnlock()
            mess = m.value
            return
        }
    }
    mux.RUnlock()

    // "NULL" is returned if no setter message is waiting
    mess = []byte("NULL")
    return
}

// creates the link between a getter and a setter
// NOTE: this implemntation means it will not work  for
//     one*getter / many*setters right now.

func crossLink(getter_id string,sender_id string) {

    if( getter_id != "" ) {

	    log.Print("crossLink link,"+getter_id+","+sender_id)
        s := link{value:sender_id}
        s.mtime=time.Now().Unix()
        // ( a MUX is used for simple locking )
        mux.Lock()
        links[getter_id] = s
        mux.Unlock()

    }
}

// remove messages where a setter has disconnected

func sendUnlink(gid string) {
    mux.Lock()
    delete(messages,gid)
    mux.Unlock()
}

// monitor go routine to provide stats in logs as needed
func monitor() {
    for {
        time.Sleep(60000*time.Millisecond);
        log.Print("no monitor defined")
    }
}

func main() {

    // Default port(s) defined
	listenPort := "6001"

    // Deal with  command line arguments
	short := "l:"
	long := []string{ "listen","manager","hub" }

    _, optargs, err := getopt.GetOpt(os.Args[1:], short, long)
	for _, oa := range optargs {
		switch oa.Opt() {
		case "-l", "--listen":
			listenPort = oa.Arg()
		}
	}

    runtype = "standalone"
	log.Print("runtype: ",runtype)

	log.Print("Listening on Port: ",listenPort);
    go monitor()

    // an incoming connection which this server recieves JSON objects from remote controller
	http.HandleFunc("/send", sendSession)

    // an incoming connection which attaches a 'controlled' session which will wait for 
    // server to send on any controller JSON objects
	http.HandleFunc("/get", getSession)

    // built in web server for all files in 'html' directory
	http.Handle("/", http.FileServer(http.Dir("./html/")))

    // Run 'ListenAndServe' 
	log.Print("Starting \n");
	err2 := http.ListenAndServe(":"+listenPort, nil)
	if err2 != nil {
         log.Print("Connection LISTEN Failed - port in use?")
	     panic("ListenAndServe: " + err.Error())
	}
}


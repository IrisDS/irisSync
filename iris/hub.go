package iris

import (
	"fmt"
	"net/http"

	"github.com/jsimnz/wsHub"
	"github.com/likexian/simplejson"
)

type Command struct {
	Cmd    string `json:"cmd,omitempty"`
	Params string `json:"params,omitempty"`
}

type Iris struct {
	clients map[string]*wsHub.Client
	hub     wsHub.WsHub
	admin   *wsHub.Client
	kill    chan bool
}

// Construct
func NewIris() Iris {
	s := Iris{
		hub:  wsHub.NewHub(),
		kill: make(chan bool),
	}
	return s
}

// Run loop
func (s Iris) Run() {

}

// Stop
func (s Iris) Kill() {
	s.kill <- true
}

// Client screen websocket connection
func (s Iris) IrisClient(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Got new screen client")
	client, err := wsHub.NewClient(w, r)
	if err != nil {
		fmt.Println("Error getting websocket connection:", err)
		return
	}

	s.hub.RegisterClient(client)
	go client.Start()
	defer func() {
		fmt.Println("Screen disconnected")
		s.hub.UnregisterClient(client)
	}()

	// Client run loop
	for {
		cmd, err := client.Read()
		if err != nil {
			fmt.Println("Couldnt read command from screen:", err)
			break
		}

		cmdjson, err := simplejson.Loads(string(cmd))
		if cmdjson.Exists("cmd") {
			cmdString, err := cmdjson.Get("cmd").String()
			if err != nil {
				fmt.Println("Invalid command from screen")
				client.WriteString("ERROR")
			} else {
				switch cmdString {

				// pause playback
				case "PAUSE":
					s.hub.Broadcast(Command{Cmd: "PAUSE"})
					break

				// start/resume playback @
				case "PLAY_AT":
					if cmdjson.Exists("at") {
						playAt, _ := cmdjson.Get("at").String()
						s.hub.Broadcast(Command{Cmd: "PLAY_AT", Params: playAt})
					} else {
						fmt.Println("Invalid play @ param")
						client.WriteJSON(Command{"INVALID_PLAY_AT"})
					}
					break
				}
			}
		}

	}
}

// Admin websocket connection
func (s Iris) IrisAdmin(w http.ResponseWriter, r *http.Request) {
	admin, err := wsHub.NewClient(w, r)
	if err != nil {
		fmt.Println("Couldnt get websocket connection")
		return
	}

	s.admin = admin
	s.hub.RegisterClient(admin)
	go s.admin.Start()

	for {
		cmdbyte, err := s.admin.Read()
		if err != nil {
			fmt.Println("Couldnt get admin command")
			continue
		}

		cmdjson, err := simplejson.Loads(string(cmdbyte))
		if err != nil {
			fmt.Println("Coudnt parse admin command")
			continue
		}

		if cmdjson.Exists("cmd") {
			switch cmd {

			// Get the devices to load a vid
			case "LOAD":
				var url string
				if cmdjson.Exists("url") {
					url, _ = cmdjson.Get("url").String()
				}
				s.hub.BroadcastJSON(Command{Cmd: "LOAD", Params: url})
				break

			// Start playback at a given point default 0
			case "PLAY_AT":

				break

			}
		} else {
			fmt.Println("Invalid command format")
		}
	}
}

// Utils

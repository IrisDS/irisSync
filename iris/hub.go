package iris

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/jsimnz/wsHub"
	"github.com/likexian/simplejson"
)

type Command struct {
	Cmd    string `json:"cmd,omitempty"`
	Params string `json:"params,omitempty"`
}

// maybe remove
type Screen struct {
	Image string
	ws    *wsHub.Client
}

type Iris struct {
	clients map[string]*Screen
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
	// watch for all device submissions
}

// Stop
func (s Iris) Kill() {
	s.kill <- true
}

// Client screen websocket connection
func (s Iris) HandleScreen(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Got new screen client")

	//Convert to ws
	client, err := wsHub.NewClient(w, r)
	if err != nil {
		fmt.Println("Error getting websocket connection:", err)
		return
	}

	//Get client ID
	vars := mux.Vars(r)
	id := vars["id"]

	// Save screen (id->ws conn)
	screen := &Screen{ws: client}
	s.clients[id] = screen

	// Register client with hub
	// Start run loop
	s.hub.RegisterClient(client)
	go client.Start()
	defer func() {
		fmt.Println("Screen disconnected")
		s.hub.UnregisterClient(client)
	}()

	// Client listen loop
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
					s.hub.BroadcastJSON(Command{Cmd: "PAUSE"})
					break

				// start/resume playback @
				case "PLAY_AT":
					if cmdjson.Exists("at") {
						playAt, _ := cmdjson.Get("at").String()
						s.hub.BroadcastJSON(Command{Cmd: "PLAY_AT", Params: playAt})
					} else {
						fmt.Println("Invalid play @ param")
					}
					break
				}
			}
		}

	}
}

// Admin websocket connection
func (s Iris) HandleAdmin(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Got new admin connection")
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
			cmd, _ := cmdjson.Get("cmd").String()

			switch cmd {

			// Get the devices to load a vid
			case "LOAD":
				var url string
				if cmdjson.Exists("url") {
					url, _ = cmdjson.Get("url").String()
				}
				for id := range s.clients {
					s.clients[id].ws.WriteJSON(Command{Cmd: "LOAD", Params: url + "/" + id})
				}
				break

			// Start playback at a given point default 0
			case "PLAY_AT":
				if cmdjson.Exists("at") {
					at, _ := cmdjson.Get("at").String()
					s.hub.BroadcastJSON(Command{Cmd: "PLAY_AT", Params: at})
				} else {
					fmt.Println("Invalid @ param for PLAY_AT")
				}
			}
		} else {
			fmt.Println("Invalid command format")
		}
	}
}

// Utils

package iris

import (
	//"fmt"
	"bytes"
	"encoding/base64"
	"image"
	"image/jpeg"
	stdlog "log"
	"net/http"
	"os"
	"strings"

	"github.com/disintegration/imaging"
	"github.com/gorilla/mux"
	"github.com/jsimnz/wsHub"
	"github.com/likexian/simplejson"
	"github.com/op/go-logging"
)

var (
	log = logging.MustGetLogger("IRIS")
)

type manifest struct {
	Nodes []manifestNodes `json:"nodes"`
}

type manifestNodes struct {
	Ip    string `json:"ip"`
	Image string `json:"image"`
}

type Command struct {
	Cmd    string      `json:"cmd,omitempty"`
	Params interface{} `json:"params,omitempty"`
}

// maybe remove
type Screen struct {
	// Image and ws conn
	Id    string
	Image image.Image
	ws    *wsHub.Client

	// Size of the screen
	width  int
	height int

	// Offset of the screen if it
	// is left or right of another
	offsetX int
	offsetY int

	left  *Screen
	right *Screen
}

type Iris struct {
	clients map[string]*Screen
	hub     wsHub.WsHub
	admin   *wsHub.Client
	kill    chan bool
	reset   chan bool

	// Keep a counter of connected/processed screens
	connectedScreens int
	processedScreens int

	isAnalyzed bool
}

func Init() {
	/** Logger **/
	logBackend := logging.NewLogBackend(os.Stderr, "[IRIS]: ", stdlog.LstdFlags|stdlog.Llongfile)
	logBackend.Color = true

	logging.SetBackend(logBackend)
	logging.SetLevel(logging.DEBUG, "IRIS")
}

// Construct
func NewIris() Iris {
	s := Iris{
		hub:        wsHub.NewHub(),
		kill:       make(chan bool),
		reset:      make(chan bool),
		clients:    make(map[string]*Screen),
		isAnalyzed: false,
	}
	return s
}

// Run loop
func (s Iris) Run() {
	// Run ws hub loop
	s.hub.Run()

	// watch for all device submissions
	for {
		select {

		// Stop the server
		case <-s.kill:
			for id := range s.clients {
				s.hub.UnregisterClient(s.clients[id].ws)
			}
			s.hub.Stop()
			break
		case <-s.reset:
			for id := range s.clients {
				s.hub.UnregisterClient(s.clients[id].ws)
			}
			s.clients = nil
			s.clients = make(map[string]*Screen)
			break
		}
	}
}

// Stop
func (s Iris) Kill() {
	s.kill <- true
}

func (i Iris) Reset() {
	i.isAnalyzed = false
	i.connectedScreens = 0
	i.processedScreens = 0
	i.reset <- true
}

// Client screen websocket connection
func (s Iris) HandleScreen(w http.ResponseWriter, r *http.Request) {
	log.Notice("Got new screen client")

	//Convert to ws
	log.Debug("Creating screen client")
	client, err := wsHub.NewClient(w, r, 512*1024, 1024)
	if err != nil {
		log.Error("Error getting websocket connection:", err)
		return
	}

	//Get client ID
	log.Debug("Parsing client ID")
	vars := mux.Vars(r)
	id := vars["id"]
	log.Debug("Client ID: %v", id)

	// Save screen (id->ws conn)
	screen := &Screen{ws: client, Id: id}
	s.clients[id] = screen
	s.connectedScreens++

	// Register client with hub
	// Start run loop
	log.Debug("Registering screen %v to wsHub", id)
	s.hub.RegisterClient(client)
	go client.Start()
	defer func() {
		log.Notice("Screen %v disconnected", id)
		s.hub.UnregisterClient(client)
	}()

	// Client listen loop
	for {
		cmd, err := client.Read()
		if err != nil {
			log.Error("Couldnt read command from screen %v: %v", id, err)
			break
		}

		// Parse command json
		cmdjson, err := simplejson.Loads(string(cmd))
		if err != nil {
			log.Error("Invalid command json for screen %v: %v", id, err)
			continue
		}

		//Make sure it fits our desired format
		if cmdjson.Exists("cmd") {
			cmdString, err := cmdjson.Get("cmd").String()

			if err != nil {
				log.Error("Invalid command from screen %v", id)
				client.WriteString("ERROR")
				continue
			}

			//log.Debug("Got command %v from screen %v", cmd, id)
			switch cmdString {

			// pause playback
			case "PAUSE":
				log.Debug("Got command PAUSE from screen %v", id)
				s.hub.BroadcastJSON(Command{Cmd: "PAUSE"})
				break

			// start/resume playback @
			case "PLAY_AT":
				log.Debug("Got command PLAY_AT from screen %v", id)
				if cmdjson.Exists("at") {
					playAt, _ := cmdjson.Get("at").Int()
					s.hub.BroadcastJSON(Command{Cmd: "PLAY_AT", Params: playAt})
				} else {
					log.Warning("Invalid play @ param from screen %v", id)
				}
				break

			// handle an image upload
			case "UPLOAD_IMAGE":
				log.Debug("Got command UPLOAD_IMAGE from screen %v", id)
				if cmdjson.Exists("width") && cmdjson.Exists("image") {
					baseImage, _ := cmdjson.Get("image").String()
					width, _ := cmdjson.Get("width").Int()
					s.clients[id].width = width

					im, err := base64ToImage(baseImage)
					if err != nil {
						log.Fatalf("Couldnt decode image from screen %v: %v", id, err)
					}
					s.clients[id].Image = im
					//time.Sleep(1 * time.Second) // Wait a second for other devices connect and start syncing
					s.processedScreens++
					log.Debug("Processed image for screen %v", id)
				} else {
					log.Error("Invalid UPLOAD_IMAGE request for screen %v", id)
				}
			}
		} else {
			log.Error("Invalid command request from sceen %v", id)
		}

	}
}

// Admin websocket connection
func (s Iris) HandleAdmin(w http.ResponseWriter, r *http.Request) {
	log.Notice("Got new admin connection")
	admin, err := wsHub.NewClient(w, r)
	if err != nil {
		log.Error("Couldnt get websocket connection for admin")
		return
	}

	s.admin = admin
	log.Debug("Registering admin")
	s.hub.RegisterClient(admin)
	log.Debug("Running admin loop")
	go s.admin.Start()

	log.Debug("Listening for admin commands")
	for {
		cmdbyte, err := s.admin.Read()
		if err != nil {
			log.Warning("Couldnt get admin command")
			continue
		}

		cmdjson, err := simplejson.Loads(string(cmdbyte))
		if err != nil {
			log.Warning("Coudnt parse admin command")
			continue
		}

		if cmdjson.Exists("cmd") {
			cmd, _ := cmdjson.Get("cmd").String()

			switch cmd {

			// Get the devices to load a vid
			case "LOAD":
				log.Debug("Instructing screens to load a resource")
				var url string
				if cmdjson.Exists("url") {
					url, _ = cmdjson.Get("url").String()
				}
				for id := range s.clients {
					s.clients[id].ws.WriteJSON(Command{Cmd: "LOAD", Params: url + "/" + id})
				}
				break

			case "ANALYZE":
				log.Debug("ANALYZING...")
				err := s.Analyze()
				if err != nil {
					log.Fatal(err)
				}
				s.PrintScreenPositions()

			// Start playback at a given point default 0
			case "PLAY_AT":
				log.Notice("Instructing screens to play @")
				if cmdjson.Exists("at") {
					at, _ := cmdjson.Get("at").Int()
					s.hub.BroadcastJSON(Command{Cmd: "PLAY_AT", Params: at})
				} else {
					log.Error("Invalid @ param for PLAY_AT")
				}
			}
		} else {
			log.Warning("Invalid command format")
		}
	}
}

// Convert base64 encoded image, into a normal jpeg
func base64ToImage(b string) (image.Image, error) {
	b = strings.Split(b, ",")[1]
	imgDec, err := base64.StdEncoding.DecodeString(b)
	if err != nil {
		return nil, err
	}
	buf := bytes.NewBuffer(imgDec)
	im, err := jpeg.Decode(buf)
	if err != nil {
		return nil, err
	}
	err = imaging.Save(im, "tmp/test1.jpg")
	if err != nil {
		return nil, err
	}

	return im, nil
}

func (i Iris) CropVideo() {

}

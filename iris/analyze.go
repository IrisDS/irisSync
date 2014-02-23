package iris

import (
	"encoding/json"
	"fmt"
	"github.com/disintegration/imaging"
	"io/ioutil"
	"os/exec"
	"sort"
)

type analyzeScore struct {
	Ip1   string  `json:"ip1"`
	Ip2   string  `json:"ip2"`
	Score float64 `json:"score"`
}

type analyze struct {
	Scr []analyzeScore `json:"scrs"`
}

// Build the manifest file, and write images to src folder
func (i Iris) Analyze() error {
	var nodes []manifestNodes
	for id := range i.clients {
		node := manifestNodes{Ip: id, Image: "img/" + id + ".jpg"}
		imaging.Save(i.clients[id].Image, "img/"+id+".jpg")
		nodes = append(nodes, node)
	}
	man := manifest{Nodes: nodes}
	buf, _ := json.Marshal(man)
	ioutil.WriteFile("manifest.json", buf, 0755)

	// Exec the iris analyze program
	out, err := exec.Command("/usr/bin/python", "/home/r2d2/Workspace/iris/src/main.py").Output()
	if err != nil {
		log.Error("Couldnt run iris analysis")
		return err
	}
	log.Debug("Finished analyzing, extracing JSON...")

	var a analyze
	err = json.Unmarshal(out, &a)
	if err != nil {
		log.Error("Couldnt get iris analysis output")
		return err
	}

	// Calculate the postions of the devices
	log.Debug("Finished iris analysis")
	log.Debug("Calculating relative postitions...")
	i.ComputePositions(a)
	return nil
}

type Guesses []*analyzeScore

func (g Guesses) Len() int      { return len(g) }
func (g Guesses) Swap(i, j int) { g[i], g[j] = g[j], g[i] }

type ByScore struct{ Guesses }

func (s ByScore) Less(i, j int) bool { return s.Guesses[i].Score < s.Guesses[j].Score }

// Now that iris has analyzed the responses
// we need to compute the postions
func (i Iris) ComputePositions(a analyze) {

	//counter := 0
	allGuesses := make([]*analyzeScore, len(i.clients))

	for k := 0; k < len(i.clients); k++ {
		allGuesses[k] = &a.Scr[k]
	}

	sort.Sort(ByScore{Guesses: allGuesses})

	for _, q := range allGuesses {
		fmt.Printf("IP1: %v, IP2: %v, Score: %v\n", q.Ip1, q.Ip2, q.Score)
	}

	/*for k := range len(i.clients) {
		var curGuess []*Screen

		for j := range len(i.clients) {

		}
	}*/

	/*for _, scorePair := range a.Scr {
		log.Debug("Position calc loop #%v", counter)
		minDist := float64(100000000)
		var minDistId string
		scr1 := i.clients[scorePair.Ip1]

		for _, scorePairN := range a.Scr {
			if ((scorePair.Ip1 != scorePairN.Ip1) && (scorePair.Ip1 == scorePairN.Ip2)) ||
				((scorePair.Ip1 == scorePairN.Ip1) && (scorePair.Ip1 != scorePairN.Ip2)) {
				score := scorePairN.Score
				if math.Abs(score) < math.Abs(minDist) {
					if scorePair.Ip1 != scorePairN.Ip1 {
						minDistId = scorePairN.Ip1
					} else {
						minDistId = scorePairN.Ip2
					}

					minDist = score
				}
			}
		}

		if minDistId != "" {
			nextscr := i.clients[minDistId]
			if minDist > 0 {
				if scr1.right != nil {
					scr1.right = nextscr
				}
				nextscr.left = scr1
				nextscr.offsetX = scr1.width
			} else {
				scr1.left = nextscr
				nextscr.right = scr1
			}
		}
		counter++
	}*/
}

func (i Iris) PrintScreenPositions() {
	/*// find the screen with 0 x offset
	var initScr *Screen
	var initId string
	for id := range i.clients {
		if i.clients[id].offsetX == 0 {
			initScr = i.clients[id]
			initId = id
			break
		}
	}

	for initScr != nil {
		fmt.Printf("Screen: %v, Height: %v, Width: %v, OffsetX: %v\n", initId, initScr.height, initScr.width, initScr.offsetX)
		initScr = initScr.right
	}*/
	for id := range i.clients {
		if i.clients[id].right != nil {
			fmt.Printf("Screen: %v, Right Screen: %v, Left Screen: %v, Height: %v, Width: %v, OffsetX: %v\n", id, i.clients[id].right.Id, "NONE", i.clients[id].height, i.clients[id].width, i.clients[id].offsetX)
		} else if i.clients[id].left != nil {
			fmt.Printf("Screen: %v, Right Screen: %v, Left Screen: %v, Height: %v, Width: %v, OffsetX: %v\n", id, "NONE", i.clients[id].left.Id, i.clients[id].height, i.clients[id].width, i.clients[id].offsetX)
		} else {
			fmt.Printf("Screen: %v, Right Screen: %v, Left Screen: %v, Height: %v, Width: %v, OffsetX: %v\n", id, "NONE", "NONE", i.clients[id].height, i.clients[id].width, i.clients[id].offsetX)
		}
	}

}

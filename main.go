package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/aaron-seo/proxy-herd/mooserver"
)

// Google Places API... should probably hide this somewhere
const KEY = "AIzaSyBDD0GRystBZTCKkgVZhgsopsF38JyH5CE"

// a string to string map of server names and their ports on SEASnet
var ports = map[string]string{
	"Juzang":  "10371",
	"Bernard": "10372",
	"Jaquez":  "10373",
	"Johnson": "10374",
	"Clark":   "10375",
}

// slice of servers that this instance will talk to (bidirectionally)
var neighbors []string

// this specific server's ID, to be parsed in command line
var serverID string

func init() {
	flag.StringVar(&serverID, "id", "default", "label to identify server")
}

func main() {
	flag.Parse()

	// set neighbors according to spec
	switch serverID {
	case "Juzang":
		neighbors = []string{"Clark", "Bernard", "Johnson"}
		break
	case "Bernard":
		neighbors = []string{"Juzang", "Jaquez", "Johnson"}
		break
	case "Jaquez":
		neighbors = []string{"Clark", "Bernard", "Johnson"}
		break
	case "Johnson":
		neighbors = []string{"Bernard", "Jaquez"}
		break
	case "Clark":
		neighbors = []string{"Jaquez", "Juzang"}
		break
	default:
		panic("invalid server id")
	}

	mux := mooserver.NewServeMux()

	// for client-server
	mux.HandleFunc("IAMAT", HandleIAMAT)
	mux.HandleFunc("WHATSAT", HandleWHATSAT)

	// for server-server
	mux.HandleFunc("AT", HandleAT)

	log.Printf("server %s listening on port %s", serverID, ports[serverID])
	log.Fatal(mooserver.ListenAndServe(":"+ports[serverID], mux))
}

// Handles IAMAT commands
func HandleIAMAT(w mooserver.ResponseWriter, r *mooserver.Request) {
	log.Printf("%s HandleIAMAT: ", serverID)

	// validate requested command
	if len(r.Command.Fields) != 4 {
		fmt.Fprintf(w, "? %s", r.Command.Raw)
		return
	}

	iamat := parseIAMAT(r.Command.Fields)
	log.Printf("%s HandleIAMAT: %+v", serverID, iamat)

	timeNow := float64(time.Now().UnixNano()) / float64(time.Second)
	timeDelta := timeNow - iamat.timestamp

	loc := location{
		serverID,
		timeDelta,
		iamat.client,
		iamat.latitude,
		iamat.longitude,
		iamat.timestamp,
	}

	storeAndPropagate(loc)

	log.Printf("%s HandleIAMAT: AT %s %f %s %f %f %f", serverID,
		serverID,
		timeDelta,
		iamat.client,
		iamat.latitude,
		iamat.longitude,
		iamat.timestamp)

	if iamat.latitude >= 0 {
		fmt.Fprintf(w, "AT %s %f %s +%f%f %f",
			serverID,
			timeDelta,
			iamat.client,
			iamat.latitude,
			iamat.longitude,
			iamat.timestamp)
	} else {
		fmt.Fprintf(w, "AT %s %f %s %f%f %f",
			serverID,
			timeDelta,
			iamat.client,
			iamat.latitude,
			iamat.longitude,
			iamat.timestamp)
	}
}

// handles WHATSAT command
func HandleWHATSAT(w mooserver.ResponseWriter, r *mooserver.Request) {
	log.Printf("%s WHATSAT", serverID)

	// validate requested command
	if len(r.Command.Fields) != 4 {
		fmt.Fprintf(w, "? %s", r.Command.Raw)
		return
	}

	locations.mu.Lock()
	defer locations.mu.Unlock()

	whatsat := parseWHATSAT(r.Command.Fields)
	log.Printf("%s WHATSAT: %+v", serverID, whatsat)

	loc := locations.lns[whatsat.client]

	query := fmt.Sprintf("https://maps.googleapis.com/maps/api/place/nearbysearch/json?location=%f,%f&radius=%f&key=%s",
		loc.latitude,
		loc.longitude,
		whatsat.radius*1000,
		KEY)

	resp, err := http.Get(query)
	if err != nil {
		log.Fatal(err)
	}

	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		log.Fatal(err)
	}

	var bodyJSON map[string]interface{}
	err = json.Unmarshal(body, &bodyJSON)
	if err != nil {
		log.Fatal(err)
	}

	// get bounded results
	tmpResult := make([]interface{}, whatsat.bound)
	for k, v := range bodyJSON {

		switch k {
		case "results":
			switch vv := v.(type) {
			case []interface{}:
				for i, u := range vv {
					if i >= whatsat.bound {
						break
					}
					tmpResult[i] = u
				}
			}
		}
	}
	bodyJSON["results"] = tmpResult
	parsedJSON, _ := json.MarshalIndent(bodyJSON, "", "\t")

	log.Printf("%s WHATSAT: AT %s %f %s %f %f %f\n%s", serverID,
		loc.serverID,
		loc.timeDelta,
		loc.client,
		loc.latitude,
		loc.longitude,
		loc.timestamp,
		string(parsedJSON))

	if loc.latitude >= 0 {
		fmt.Fprintf(w, "AT %s %f %s +%f%f %f\n%s",
			loc.serverID,
			loc.timeDelta,
			loc.client,
			loc.latitude,
			loc.longitude,
			loc.timestamp,
			string(parsedJSON))
	} else {
		fmt.Fprintf(w, "AT %s %f %s %f%f %f\n%s",
			loc.serverID,
			loc.timeDelta,
			loc.client,
			loc.latitude,
			loc.longitude,
			loc.timestamp,
			string(parsedJSON))
	}
}

// handles AT command
func HandleAT(w mooserver.ResponseWriter, r *mooserver.Request) {
	log.Printf("%s AT", serverID)

	// validate requested command
	if len(r.Command.Fields) != 6 {
		fmt.Fprintf(w, "? %s", r.Command.Raw)
		return
	}

	at := parseAT(r.Command.Fields)
	log.Printf("%s AT: %+v", serverID, at)

	loc := location{
		at.serverID,
		at.timeDelta,
		at.client,
		at.latitude,
		at.longitude,
		at.timestamp,
	}
	storeAndPropagate(loc)
}

// helper functions to parse commands into respective data structures
func parseIAMAT(fields []string) IAMAT {
	latitude, longitude := parseCoordinate(fields[2])

	timestamp, _ := strconv.ParseFloat(fields[3], 64)

	parsed := IAMAT{
		client:    fields[1],
		latitude:  latitude,
		longitude: longitude,
		timestamp: timestamp,
	}
	return parsed
}

func parseWHATSAT(fields []string) WHATSAT {
	radius, _ := strconv.ParseFloat(fields[2], 64)
	bound, _ := strconv.Atoi(fields[3])

	parsed := WHATSAT{
		client: fields[1],
		radius: radius,
		bound:  bound,
	}
	return parsed
}

func parseAT(fields []string) AT {
	latitude, longitude := parseCoordinate(fields[4])
	timeDelta, _ := strconv.ParseFloat(fields[2], 64)
	timestamp, _ := strconv.ParseFloat(fields[5], 64)

	parsed := AT{
		fields[1],
		timeDelta,
		fields[3],
		latitude,
		longitude,
		timestamp,
	}
	return parsed
}

type IAMAT struct {
	client    string
	latitude  float64
	longitude float64
	timestamp float64
}

type WHATSAT struct {
	client string
	radius float64
	bound  int
}

type AT struct {
	serverID  string
	timeDelta float64
	client    string
	latitude  float64
	longitude float64
	timestamp float64
}

// helper function for ISO coordinates
func parseCoordinate(coord string) (float64, float64) {
	ISOCoord := regexp.MustCompile(`((\+|-)\d+\.?\d*){2}`)
	result := ISOCoord.FindString(coord)
	INDCoord := regexp.MustCompile(`(\+|-)\d+\.?\d*`)

	pair := INDCoord.FindAllString(result, 2)
	latitude, _ := strconv.ParseFloat(pair[0], 64)
	longitude, _ := strconv.ParseFloat(pair[1], 64)

	return latitude, longitude
}

// for storing locations in server
type location struct {
	serverID  string
	timeDelta float64
	client    string
	latitude  float64
	longitude float64
	timestamp float64
}

type locationsStore struct {
	mu  sync.RWMutex
	lns map[string]location
}

var locations locationsStore

func storeAndPropagate(loc location) {
	if ok := check(loc); ok {
		store(loc)
		propagate(loc)
	} else {
		//log.Println("circular")
	}

}

// checks circular cycling
func check(loc location) bool {
	locations.mu.Lock()
	defer locations.mu.Unlock()
	if _, ok := locations.lns[loc.client]; ok {
		if locations.lns[loc.client].timestamp == loc.timestamp {
			return false
		}
	}
	return true
}

func store(loc location) {
	locations.mu.Lock()
	defer locations.mu.Unlock()

	if locations.lns == nil {
		locations.lns = make(map[string]location)
	}
	locations.lns[loc.client] = loc
}

func propagate(loc location) {
	for _, neighbor := range neighbors {
		//log.Println("neighbor " + neighbor)
		conn, err := net.Dial("tcp", "localhost:"+ports[neighbor])
		if err != nil {
			log.Printf("%s Error dialing tcp to %s", serverID, neighbor)
			return
		}
		defer conn.Close()

		if loc.latitude >= 0 {
			//log.Printf("IAMAT %s +%f%f %f", loc.client, loc.latitude, loc.longitude, loc.timestamp)
			fmt.Fprintf(conn, "AT %s %f %s +%f%f %f", loc.serverID, loc.timeDelta, loc.client, loc.latitude, loc.longitude, loc.timestamp)
		} else {
			fmt.Fprintf(conn, "AT %s %f %s %f%f %f", loc.serverID, loc.timeDelta, loc.client, loc.latitude, loc.longitude, loc.timestamp)
		}
	}
}

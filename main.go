package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/aaron-seo/proxy-herd/mooserver"
)

// a string to string map of server names and their ports on SEASnet
var ports = map[string]string{
	"Juzang":  "10371",
	"Bernard": "10372",
	"Jaquez":  "10373",
	"Johnson": "10374",
	"Clark":   "10375",
}

var neighbors []string

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

	mux.HandleFunc("IAMAT", HandleIAMAT)
	mux.HandleFunc("WHATSAT", HandleWHATSAT)

	log.Printf("server %s listening on port %s", serverID, ports[serverID])
	log.Fatal(mooserver.ListenAndServe(":"+ports[serverID], mux))
}

func HandleIAMAT(w mooserver.ResponseWriter, r *mooserver.Request) {
	log.Println("HandleIAMAT")
	iamat := parseIAMAT(r.Command.Fields)

	timeNow := float64(time.Now().UnixNano()) / float64(time.Second)
	timeDelta := timeNow - iamat.timestamp

	// TODO validate/sanitize data

	loc := location{
		serverID,
		timeDelta,
		iamat.client,
		iamat.latitude,
		iamat.longitude,
		iamat.timestamp,
	}
	storeAndPropagate(loc)

	fmt.Fprintf(w, "AT %s %f %s %f %f %f",
		serverID,
		timeDelta,
		iamat.client,
		iamat.latitude,
		iamat.longitude,
		iamat.timestamp)
}

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
	store(loc)
	propagate(loc)
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
		log.Println("neighbor " + neighbor)
		conn, err := net.Dial("tcp", "localhost:"+ports[neighbor])
		if err != nil {
			log.Println("error dialing tcp to neighbor")
			return
		}

		if loc.latitude >= 0 {
			log.Printf("IAMAT %s +%f%f %f", loc.client, loc.latitude, loc.longitude, loc.timestamp)
			fmt.Fprintf(conn, "IAMAT %s +%f%f %f", loc.client, loc.latitude, loc.longitude, loc.timestamp)
		} else {
			fmt.Fprintf(conn, "IAMAT %s -%f%f %f", loc.client, loc.latitude, loc.longitude, loc.timestamp)
		}
	}
}

func HandleWHATSAT(w mooserver.ResponseWriter, r *mooserver.Request) {
	locations.mu.Lock()
	defer locations.mu.Unlock()

	fmt.Fprintf(w, "%+v", locations.lns[r.Command.Fields[1]])
}

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

func parseCoordinate(coord string) (float64, float64) {
	ISOCoord := regexp.MustCompile(`((\+|-)\d+\.?\d*){2}`)
	result := ISOCoord.FindString(coord)
	INDCoord := regexp.MustCompile(`(\+|-)\d+\.?\d*`)

	pair := INDCoord.FindAllString(result, 2)
	latitude, _ := strconv.ParseFloat(pair[0], 64)
	longitude, _ := strconv.ParseFloat(pair[1], 64)

	return latitude, longitude
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

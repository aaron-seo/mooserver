# mooserver
A proxy TCP server herd architecture prototype for Google Places API written in Go.

Based on Dr. Paul Eggert's CS131 Project Spec (original project calls for Python).

### Usage
`go run main.go --id [idname]`

(e.g. `go run main.go --id Clark`)

## Credits
Fundamental server architecture takes inspiration from the standard library's net/http.

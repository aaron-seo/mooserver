# mooserver
A proxy TCP server herd architecture prototype for Google Places API written in Go.

Based on Dr. Paul Eggert's CS131 Project Spec (original project calls for Python).

### 1. Build
`go build`

### 2. Start a mooserver instance
`./proxy-herd --id [idname]`

(e.g. `./proxy-herd --id Clark`)

## Credits
Fundamental server architecture takes inspiration from the standard library's net/http.

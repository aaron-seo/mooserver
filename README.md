# mooserver
A proxy TCP server herd architecture prototype for Google Places API written in Go.

Based on Dr. Paul Eggert's CS131 Project Spec (original project calls for Python; I used Go with permission from TA).

### Usage
The binary that comes in the submission tar should be able to run on SEASnet servers (I tested on lnxsrv15).

You can use do the following to start a server:

`./proxy-herd --id [idname]`

(e.g. `./proxy-herd --id Clark`)

I was able to make some tweaks to the provided TA testing client and the servers ran fine with the script, too.

Namely, I adjusted the set_server_info() and start_server() functions to match the above command.

`self.server = os.path.join(server_dir, "./proxy-herd")`
`command = 'nohup {} --id {} &\n'.format(self.server, self.server_name)`

(Please contact me at aaronseo@ucla.edu if there are any issues).


If your environment has Go installed (lnxsrv15 on SEASnet does not seem to), you can run using `go run`

`go run main.go --id [idname]`

(e.g. `go run main.go --id Clark`)

Alternatively, you can build for your environment and run the binary.

`go build`

`./proxy-herd --id [idname]`


## Credits
Fundamental server architecture takes inspiration from the standard library's net/http.

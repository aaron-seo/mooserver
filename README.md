# mooserver
A proxy TCP server herd architecture prototype for Google Places API written in Go.

Based on Dr. Paul Eggert's CS131 Project Spec (original project calls for Python; I used Go with permission from TA).

### Usage

#### Starting a server

If your environment has Go installed you can run using `go run`.

`go run main.go --id [idname]`

(e.g. `go run main.go --id Clark`)

Alternatively, you can build for your environment and run the binary.

`go build`

`./proxy-herd --id [idname]`

#### Sending commands to server (from project spec)

Your prototype should consist of five servers (with server IDs 'Juzang', 'Bernard', 'Jaquez', 'Johnson', 'Clark') that communicate to each other (bidirectionally) with the following pattern:

    Clark talks with Jaquez and Juzang.
    Bernard talks with everyone else but Clark.
    Juzang talks with Johnson.

Each server should accept TCP connections from clients that emulate mobile devices with IP addresses and DNS names. A client should be able to send its location to the server by sending a message using this format:

IAMAT kiwi.cs.ucla.edu +34.068930-118.445127 1621464827.959498503

The first field IAMAT is name of the command where the client tells the server where it is. Its operands are the client ID (in this case, kiwi.cs.ucla.edu), the latitude and longitude in decimal degrees using ISO 6709 notation, and the client's idea of when it sent the message, expressed in POSIX time, which consists of seconds and nanoseconds since 1970-01-01 00:00:00 UTC, ignoring leap seconds; for example, 1621464827.959498503 stands for 2021-05-19 22:53:47.959498503 UTC. A client ID may be any string of non-white-space characters. (A white space character is space, tab, carriage return, newline, formfeed, or vertical tab.) Fields are separated by one or more white space characters and do not contain white space; ignore any leading or trailing white space on the line.

The server should respond to clients with a message using this format:

AT Clark +0.263873386 kiwi.cs.ucla.edu +34.068930-118.445127 1621464827.959498503

where AT is the name of the response, Clark is the ID of the server that got the message from the client, +0.263873386 is the difference between the server's idea of when it got the message from the client and the client's time stamp, and the remaining fields are a copy of the IAMAT data. In this example (the normal case), the server's time stamp is greater than the client's so the difference is positive, but it might be negative if there was enough clock skew in that direction.

Clients can also query for information about places near other clients' locations, with a query using this format:

WHATSAT kiwi.cs.ucla.edu 10 5

The arguments to a WHATSAT message are the name of another client (e.g., kiwi.cs.ucla.edu), a radius (in kilometers) from the client (e.g., 10), and an upper bound on the amount of information to receive from Places data within that radius of the client (e.g., 5). The radius must be at most 50 km, and the information bound must be at most 20 items, since that's all that the Places API supports conveniently.

The server responds with a AT message in the same format as before, giving the most recent location reported by the client, along with the server that it talked to and the time the server did the talking. Following the AT message is a JSON-format message, exactly in the same format that Google Places gives for a Nearby Search request (except that any sequence of two or more adjacent newlines is replaced by a single newline and that all trailing newlines are removed), followed by two newlines. Here is an example (with some details omitted and replaced with "...").

AT Clark +0.263873386 kiwi.cs.ucla.edu +34.068930-118.445127 1621464827.959498503
{
   "html_attributions" : [],
   "next_page_token" : "CvQ...L2E",
   "results" : [
      {
         "geometry" : {
            "location" : {
               "lat" : 34.068921,
               "lng" : -118.445181
            }
         },
         "icon" : "http://maps.gstatic.com/mapfiles/place_api/icons/university-71.png",
         "id" : "4d56f16ad3d8976d49143fa4fdfffbc0a7ce8e39",
         "name" : "University of California, Los Angeles",
         "photos" : [
            {
               "height" : 1200,
               "html_attributions" : [ "From a Google User" ],
               "photo_reference" : "CnR...4dY",
               "width" : 1600
            }
         ],
         "rating" : 4.5,
         "reference" : "CpQ...r5Y",
         "types" : [ "university", "establishment" ],
         "vicinity" : "Los Angeles"
      },
      ...
   ],
   "status" : "OK"
}



## Credits
Server architecture takes inspiration from the standard library's net/http.

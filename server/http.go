package server

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	eventsource "github.com/antage/eventsource/http"
	"github.com/gorilla/websocket"
)

// Connection represents a websocket connection along with reader and writer state
type Connection struct {
	ws          *websocket.Conn
	es          eventsource.EventSource
	send        chan []byte
	isWebsocket bool
}

func (c *Connection) esWriter(f *Server) {
	fmt.Println("Writer started.")
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		c.es.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				c.esWrite([]byte{})
				return
			}
			if err := c.esWrite([]byte(message)); err != nil {
				return
			}
		case <-ticker.C:
			fmt.Println("tick.")
			if err := c.esWrite([]byte{}); err != nil {
				return
			}
		}
	}
}

func (c *Connection) esWrite(payload []byte) error {
	fmt.Println("Writing to eventsource: ", string(payload))
	c.es.SendMessage(string(payload), "", "")
	return nil
}

/*
from faye js:

handle: function(request, response) {
    var requestUrl    = url.parse(request.url, true),
        requestMethod = request.method,
        origin        = request.headers.origin,
        self          = this;

    request.originalUrl = request.url;

    request.on('error', function(error) { self._returnError(response, error) });
    response.on('error', function(error) { self._returnError(null, error) });

    if (this._static.test(requestUrl.pathname))
      return this._static.call(request, response);

    // http://groups.google.com/group/faye-users/browse_thread/thread/4a01bb7d25d3636a
    if (requestMethod === 'OPTIONS' || request.headers['access-control-request-method'] === 'POST')
      return this._handleOptions(response);

    if (Faye.EventSource.isEventSource(request))
      return this.handleEventSource(request, response);

    if (requestMethod === 'GET')
      return this._callWithParams(request, response, requestUrl.query);

    if (requestMethod === 'POST')
      return Faye.withDataFor(request, function(data) {
        var type   = (request.headers['content-type'] || '').split(';')[0],
            params = (type === 'application/json')
                   ? {message: data}
                   : querystring.parse(data);

        request.body = data;
        self._callWithParams(request, response, params);
      });

    this._returnError(response, {message: 'Unrecognized request type'});
  },


 _handleOptions: function(response) {
    var headers = {
      'Access-Control-Allow-Credentials': 'false',
      'Access-Control-Allow-Headers':     'Accept, Content-Type, Pragma, X-Requested-With',
      'Access-Control-Allow-Methods':     'POST, GET, PUT, DELETE, OPTIONS',
      'Access-Control-Allow-Origin':      '*',
      'Access-Control-Max-Age':           '86400'
    };
    response.writeHead(200, headers);
    response.end('');
  },
*/

func serveOther(w http.ResponseWriter, r *http.Request) {
	fmt.Println("serve other: ", r.URL)

	fmt.Println("REQUEST URL: ", r.URL.Path)
	fmt.Println("REQUEST RAW QUERY: ", r.URL.RawQuery)
	fmt.Println("REQUEST HEADER ", r.Header)

	if isEventSource(r) {
		fmt.Println("Looks like event source")
		handleEventSource(w, r)
	}
	w.WriteHeader(http.StatusOK)
	return
}

func handleEventSource(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Handle event source: ", r.URL.Path)
	// create a new connection for the event source action
	clientID := strings.Split(r.URL.Path, "/")[2]
	fmt.Println("clientID: ", clientID)
	es := eventsource.New(nil, nil)
	c := &Connection{send: make(chan []byte, 256), es: es, isWebsocket: false}
	// TODO: NEED TO ASSOCIATED THE EXISTING FAYE CLIENT INFO/SUBSCRIPTIONS WITH THE CONNECTION CHANNEL
	// USE CLIENT ID TO UPDATE FAYE INFO WITH ES CONNETION CHANNEL
	f.UpdateClientChannel(clientID, c.send)
	go c.esWriter(f)
	c.es.ServeHTTP(w, r)
	return
}

/*
handleEventSource: function(request, response) {
    var es       = new Faye.EventSource(request, response, {ping: this._options.ping}),
        clientId = es.url.split('/').pop(),
        self     = this;

    this.debug('Opened EventSource connection for ?', clientId);
    this._server.openSocket(clientId, es, request);

    es.onclose = function(event) {
      self._server.closeSocket(clientId);
      es = null;
    };
  },
*/

/*
serverWs - provides an http handler for upgrading a connection to a websocket connection
*/
func serveWs(w http.ResponseWriter, r *http.Request) {
	fmt.Println("serveWs")
	fmt.Println("METHOD: ", r.Method)
	fmt.Println("REQUEST URL: ", r.URL.Path)
	fmt.Println("REQUEST RAW QUERY: ", r.URL.RawQuery)
	fmt.Println("REQUEST HEADER ", r.Header)

	// handle options
	// if r.Method == "OPTIONS" || r.Header.Get("Access-Control-Request-Method") == "POST" {
	// 	handleOptions(w, r)
	// }

	// if isEventSource(r) {
	// 	fmt.Println("Is event source")
	// }

	// if r.Method != "GET" {
	// 	http.Error(w, "Method not allowed", 405)
	// 	//serveLongPolling(f, w, r)
	// 	return
	// }

	/*
	   if r.Header.Get("Origin") != "http://"+r.Host {
	           http.Error(w, "Origin not allowed", 403)
	           return
	   }
	*/

	ws, err := websocket.Upgrade(w, r, nil, 1024, 1024)
	if _, ok := err.(websocket.HandshakeError); ok {
		//http.Error(w, "Not a websocket handshake", 400)
		fmt.Println("ERR:", err)
		fmt.Println("NOT A WEBSOCKET HANDSHAKE")
		serveJSONP(f, w, r)
		return
	} else if err != nil {
		fmt.Println(err)
		return
	}
	c := &Connection{send: make(chan []byte, 256), ws: ws, isWebsocket: true}
	go c.writer(f)
	c.reader(f)
}

/*
handleOptions allows for access control awesomeness
*/
func handleOptions(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Handle options!")
	w.Header().Set("Access-Control-Allow-Credentials", "false")
	w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Pragma, X-Requested-With")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Max-Age", "86400")
	w.WriteHeader(http.StatusOK)
	return
}

// EventSource.isEventSource = function(request) {
//   if (request.method !== 'GET') return false;
//   var accept = (request.headers.accept || '').split(/\s*,\s*/);
//   return accept.indexOf('text/event-stream') >= 0;
// };
/*
isEventSource
*/
func isEventSource(r *http.Request) bool {
	fmt.Println("isEventSource? ", r.Method)
	if r.Method != "GET" {
		return false
	}

	accept := r.Header.Get("Accept")
	fmt.Println("Accept: ", accept)
	return accept == "text/event-stream"
}

var f *Server

// Start inits the http server on the address/port given in the addr param
func Start(addr string) {
	f = NewServer()
	http.HandleFunc("/bayeux", serveWs)
	http.HandleFunc("/", serveOther)

	// serve static assets workaround
	//http.Handle("/file/", http.StripPrefix("/file", http.FileServer(http.Dir("/Users/paul/go/src/github.com/pcrawfor/fayego/runner"))))

	err := http.ListenAndServe(addr, nil)
	if err != nil {
		fmt.Println("Fatal error ", err.Error())
		os.Exit(1)
	}
}

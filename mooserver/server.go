package mooserver

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
)

type Server struct {
	Addr string

	Handler Handler

	mu sync.RWMutex
}

func (s *Server) Serve(ln net.Listener) error {
	defer ln.Close()
	for {
		rwc, err := ln.Accept()
		if err != nil {
			return errors.New("error accepting connection")
		}

		c := s.newConn(rwc)

		go c.serve()
	}
}

func (s *Server) ListenAndServe() error {
	ln, err := net.Listen("tcp", s.Addr)
	if err != nil {
		log.Fatal(err)
	}
	return s.Serve(ln)
}

func ListenAndServe(address string, handler Handler) error {
	server := &Server{
		Addr:    address,
		Handler: handler,
	}
	return server.ListenAndServe()
}

func (s *Server) newConn(rwc net.Conn) *conn {
	c := &conn{
		server: s,
		rwc:    rwc,
	}
	return c
}

type conn struct {
	server *Server
	rwc    net.Conn
}

func (c *conn) readRequest() *Request {
	scanner := bufio.NewScanner(c.rwc)

	var data string
	scanner.Scan()
	data = scanner.Text()

	cmd := parseCommand(data)

	req := &Request{
		Command: cmd,
	}
	return req
}

func (c *conn) serve() {
	req := c.readRequest()
	c.server.Handler.Serve(c.rwc, req)
}

// A Handler works with a request to a server
type Handler interface {
	Serve(ResponseWriter, *Request)
}

type HandlerFunc func(ResponseWriter, *Request)

func (f HandlerFunc) Serve(w ResponseWriter, r *Request) {
	f(w, r)
}

type ResponseWriter interface {
	Write([]byte) (int, error)
}

type Request struct {
	Command command
}

type command struct {
	Method string
	Fields []string // fields[0] == method
	Raw    string
}

// Helper functions for parsing commands
func parseCommand(rawCommand string) command {
	fields := strings.Fields(rawCommand)
	cmd := command{
		Method: fields[0],
		Fields: fields,
		Raw:    rawCommand,
	}
	return cmd
}

type ServeMux struct {
	mu    sync.RWMutex
	m     map[string]muxEntry
	es    []muxEntry
	hosts bool
}

type muxEntry struct {
	h       Handler
	pattern string
}

func NewServeMux() *ServeMux {
	return new(ServeMux)
}

// naive matching
func (mux *ServeMux) match(method string) (h Handler, pattern string) {
	v, ok := mux.m[method]
	if ok {
		return v.h, v.pattern
	}

	return HandlerFunc(noMatch), ""
}

// when no match for a requested command in the mux
func noMatch(w ResponseWriter, r *Request) {
	fmt.Fprintf(w, "? %s", r.Command.Raw)
}

// matches a command to mux (if successful) and serves the request
func (mux *ServeMux) Serve(w ResponseWriter, r *Request) {
	h, _ := mux.match(r.Command.Method)
	h.Serve(w, r)
}

// Registers a command into mux
func (mux *ServeMux) Handle(pattern string, handler Handler) {
	mux.mu.Lock()
	defer mux.mu.Unlock()

	if _, exists := mux.m[pattern]; exists {
		panic("sheepsrv: multiple registrations for " + pattern)
	}

	if mux.m == nil {
		mux.m = make(map[string]muxEntry)
	}

	entry := muxEntry{
		h:       handler,
		pattern: pattern,
	}

	mux.m[pattern] = entry
}

// wrapper for HandlerFunc so we can have ordinary Go functions as Handlers
func (mux *ServeMux) HandleFunc(pattern string, handler func(ResponseWriter, *Request)) {
	if handler == nil {
		panic("nil handler")
	}

	mux.Handle(pattern, HandlerFunc(handler))
}

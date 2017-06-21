// Package martini is a powerful package for quickly writing modular web applications/services in Golang.
//
// For a full guide visit http://github.com/go-martini/martini
//
//  package main
//
//  import "github.com/go-martini/martini"
//
//  func main() {
//    m := martini.Classic()
//
//    m.Get("/", func() string {
//      return "Hello world!"
//    })
//
//    m.Run()
//  }
package martini

import (
	"log"
	"net/http"
	"os"
	"reflect"

	"github.com/codegangsta/inject"
	"net"
	"time"
	"sync"
)

// Martini represents the top level web application. inject.Injector methods can be invoked to map services on a global level.
type Martini struct {
	inject.Injector
	handlers []Handler
	action   Handler
	logger   *log.Logger
	server MartiniServer
}

type MartiniServer struct {
	listener net.Listener
	connStateChan chan *ConnState
	conns map[net.Conn]*ConnState
	connsLock sync.Mutex
}

type ConnState struct {
	conn net.Conn
	state http.ConnState
}

// New creates a bare bones Martini instance. Use this method if you want to have full control over the middleware that is used.
func New() *Martini {
	m := &Martini{Injector: inject.New(), action: func() {}, logger: log.New(os.Stdout, "[martini] ", 0)}
	m.server.connStateChan = make(chan *ConnState, 2048)
	m.server.conns = make(map[net.Conn]*ConnState)
	m.Map(m.logger)
	m.Map(defaultReturnHandler())
	return m
}

// Handlers sets the entire middleware stack with the given Handlers. This will clear any current middleware handlers.
// Will panic if any of the handlers is not a callable function
func (m *Martini) Handlers(handlers ...Handler) {
	m.handlers = make([]Handler, 0)
	for _, handler := range handlers {
		m.Use(handler)
	}
}

// Action sets the handler that will be called after all the middleware has been invoked. This is set to martini.Router in a martini.Classic().
func (m *Martini) Action(handler Handler) {
	validateHandler(handler)
	m.action = handler
}

// Logger sets the logger
func (m *Martini) Logger(logger *log.Logger) {
	m.logger = logger
	m.Map(m.logger)
}

// Use adds a middleware Handler to the stack. Will panic if the handler is not a callable func. Middleware Handlers are invoked in the order that they are added.
func (m *Martini) Use(handler Handler) {
	validateHandler(handler)

	m.handlers = append(m.handlers, handler)
}

// ServeHTTP is the HTTP Entry point for a Martini instance. Useful if you want to control your own HTTP server.
func (m *Martini) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	m.createContext(res, req).run()
}

// Run the http server on a given host and port.
func (m *Martini) RunOnAddr(addr string) {
	// TODO: Should probably be implemented using a new instance of http.Server in place of
	// calling http.ListenAndServer directly, so that it could be stored in the martini struct for later use.
	// This would also allow to improve testing when a custom host and port are passed.

	logger := m.Injector.Get(reflect.TypeOf(m.logger)).Interface().(*log.Logger)
	logger.Printf("listening on %s (%s)\n", addr, Env)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		logger.Fatalln(err)
	}
	m.server.listener = listener
	server := &http.Server{Addr:addr, Handler: m, ConnState: m.ConnStateHook}
	go func () {
		connStateChan := m.server.connStateChan
		for connState := range connStateChan {
			switch connState.state {
			case http.StateNew, http.StateActive, http.StateIdle:
				m.server.connsLock.Lock()
				m.server.conns[connState.conn] = connState
				m.server.connsLock.Unlock()
			case http.StateHijacked, http.StateClosed:
				m.server.connsLock.Lock()
				delete(m.server.conns, connState.conn)
				m.server.connsLock.Unlock()
			}
		}
	}()
	server.Serve(listener)
}

// Run the http server. Listening on os.GetEnv("PORT") or 3000 by default.
func (m *Martini) Run() {
	port := os.Getenv("PORT")
	if len(port) == 0 {
		port = "3000"
	}

	host := os.Getenv("HOST")

	m.RunOnAddr(host + ":" + port)
}

func (m *Martini) closeIdleConn() {
	m.server.connsLock.Lock()
	for _, connState := range m.server.conns {
		if connState.state == http.StateIdle {
			connState.conn.Close()
		}
	}
	m.server.connsLock.Unlock()
}

func (m *Martini) Close() {
	if m.server.listener != nil {
		m.server.listener.Close()
		m.server.listener = nil
	}
	for {
		if len(m.server.conns) == 0 {
			break
		}
		m.closeIdleConn()
		time.Sleep(500 * time.Millisecond)
	}
	if m.server.connStateChan != nil {
		close(m.server.connStateChan)
		m.server.connStateChan = nil
	}
}

func (m *Martini) ConnStateHook(conn net.Conn, state http.ConnState) {
	connState := &ConnState{
		conn: conn,
		state: state,
	}
	select {
	case m.server.connStateChan <- connState:
	default:
		logger := m.Injector.Get(reflect.TypeOf(m.logger)).Interface().(*log.Logger)
		logger.Printf("martini connStateChan block, conn:%s, state:%d\n", conn.RemoteAddr(), state)
	}
}

func (m *Martini) createContext(res http.ResponseWriter, req *http.Request) *context {
	c := &context{inject.New(), m.handlers, m.action, NewResponseWriter(res), 0}
	c.SetParent(m)
	c.MapTo(c, (*Context)(nil))
	c.MapTo(c.rw, (*http.ResponseWriter)(nil))
	c.Map(req)
	return c
}

// ClassicMartini represents a Martini with some reasonable defaults. Embeds the router functions for convenience.
type ClassicMartini struct {
	*Martini
	Router
}

// Classic creates a classic Martini with some basic default middleware - martini.Logger, martini.Recovery and martini.Static.
// Classic also maps martini.Routes as a service.
func Classic() *ClassicMartini {
	r := NewRouter()
	m := New()
	m.Use(Logger())
	m.Use(Recovery())
	m.Use(Static("public"))
	m.MapTo(r, (*Routes)(nil))
	m.Action(r.Handle)
	return &ClassicMartini{m, r}
}

// Handler can be any callable function. Martini attempts to inject services into the handler's argument list.
// Martini will panic if an argument could not be fullfilled via dependency injection.
type Handler interface{}

func validateHandler(handler Handler) {
	if reflect.TypeOf(handler).Kind() != reflect.Func {
		panic("martini handler must be a callable func")
	}
}

// Context represents a request context. Services can be mapped on the request level from this interface.
type Context interface {
	inject.Injector
	// Next is an optional function that Middleware Handlers can call to yield the until after
	// the other Handlers have been executed. This works really well for any operations that must
	// happen after an http request
	Next()
	// Written returns whether or not the response for this context has been written.
	Written() bool
}

type context struct {
	inject.Injector
	handlers []Handler
	action   Handler
	rw       ResponseWriter
	index    int
}

func (c *context) handler() Handler {
	if c.index < len(c.handlers) {
		return c.handlers[c.index]
	}
	if c.index == len(c.handlers) {
		return c.action
	}
	panic("invalid index for context handler")
}

func (c *context) Next() {
	c.index += 1
	c.run()
}

func (c *context) Written() bool {
	return c.rw.Written()
}

func (c *context) run() {
	for c.index <= len(c.handlers) {
		_, err := c.Invoke(c.handler())
		if err != nil {
			panic(err)
		}
		c.index += 1

		if c.Written() {
			return
		}
	}
}

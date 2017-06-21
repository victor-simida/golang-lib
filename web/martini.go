package web

import (
	"fmt"
	"github.com/go-martini/martini"
	"github.com/martini-contrib/cors"
	"github.com/martini-contrib/gzip"
	"github.com/martini-contrib/render"
)

type regInfo struct {
	Method  string
	Uri     string
	Handler []martini.Handler
}

var _RegInfo []regInfo = make([]regInfo, 0, 50)
var _Middleware []martini.Handler = make([]martini.Handler, 0, 10)

func RegisterHandler(method, uri string, handler ...martini.Handler) {
	info := regInfo{method, uri, handler}
	_RegInfo = append(_RegInfo, info)
}

func RegisterMiddleware(m martini.Handler) {
	_Middleware = append(_Middleware, m)
}

var m *martini.ClassicMartini

func RunMartini() {
	m = martini.Classic()
	m.Handlers(martini.Recovery(), martini.Static("public"))
	m.Use(gzip.All())
	m.Use(martini.Static("_H5")) //add to H5
	m.Use(render.Renderer(render.Options{
		Directory:  "_H5",
		Layout:     "index",
		Extensions: []string{".html"},
	}))
	m.Use(cors.Allow(&cors.Options{
		AllowAllOrigins:  true,
		AllowMethods:     []string{"PUT", "GET", "POST", "DELETE", "OPTIONS"},
		ExposeHeaders:    []string{"Content-Length","versionNo","aopsid"},
		AllowHeaders:     []string{"Content-Length","versionNo","aopsid"},
		AllowCredentials: true,
	}))
	for _, mid := range _Middleware {
		m.Use(mid)
	}

	for _, info := range _RegInfo {
		switch info.Method {
		case "Get":
			m.Get(info.Uri, info.Handler...)
		case "Post":
			m.Post(info.Uri, info.Handler...)
		}
	}

	port := ""
	port = "9999"

	m.RunOnAddr(`:` + port)
}

func CloseMartini() {
	m.Close()
	fmt.Println("Close Martini")

}

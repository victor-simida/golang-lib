package web

import (
	"github.com/go-martini/martini"
	"golang-lib/mylog"
	"violate/status"
	"net/http"
	"time"
	"github.com/martini-contrib/render"
)

type TraceInfo struct {
	TraceId string
}

func (this *TraceInfo) ToString() string {
	return this.TraceId
}

var SetTraceId = func(w http.ResponseWriter, r *http.Request, c martini.Context, ren render.Render) {
	t := new(status.Status_t)
	c.Map(t)
	now := time.Now()
	t.Init(r)
	ren.SetTraceStatus(t)

	c.Next() /*通过c next来判断流程是否已经走完*/
	cost := time.Since(now)
	t.SetEndTimeWithNow()
	t.SetDurMillis(int64(cost))
	t.WriteInParam(r.Form)
	mylog.LOG.I("SetTraceId Return %s", t.ToString())
	t.SendToKafka()
	t = nil
}

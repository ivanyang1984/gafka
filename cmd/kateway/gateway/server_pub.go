// +build !fasthttp

package gateway

import (
	"net"
	"net/http"
	"os"
	"time"

	"github.com/funkygao/golib/ratelimiter"
	log "github.com/funkygao/log4go"
)

type pubServer struct {
	*webServer

	pubMetrics  *pubMetrics
	throttlePub *ratelimiter.LeakyBuckets
	auditor     log.Logger

	throttleBadAppid *ratelimiter.LeakyBuckets
}

func newPubServer(httpAddr, httpsAddr string, maxClients int, gw *Gateway) *pubServer {
	this := &pubServer{
		webServer:        newWebServer("pub_server", httpAddr, httpsAddr, maxClients, Options.HttpReadTimeout, gw),
		throttlePub:      ratelimiter.NewLeakyBuckets(Options.PubQpsLimit, time.Minute),
		throttleBadAppid: ratelimiter.NewLeakyBuckets(3, time.Minute),
	}
	this.pubMetrics = NewPubMetrics(this.gw)
	this.onConnNewFunc = this.onConnNew
	this.onConnCloseFunc = this.onConnClose

	this.webServer.onStop = func() {
		this.pubMetrics.Flush()
	}

	this.auditor = log.NewDefaultLogger(log.TRACE)
	this.auditor.DeleteFilter("stdout")

	_ = os.Mkdir("audit", os.ModePerm)
	rotateEnabled, discardWhenDiskFull := true, false
	filer := log.NewFileLogWriter("audit/pub_audit.log", rotateEnabled, discardWhenDiskFull, 0644)
	if filer == nil {
		panic("failed to open pub audit log")
	}
	filer.SetFormat("[%d %T] [%L] (%S) %M")
	if Options.LogRotateSize > 0 {
		filer.SetRotateSize(Options.LogRotateSize)
	}
	filer.SetRotateLines(0)
	filer.SetRotateDaily(true)
	this.auditor.AddFilter("file", logLevel, filer)

	return this
}

func (this *pubServer) Start() {
	this.pubMetrics.Load()
	this.webServer.Start()
}

func (this *pubServer) onConnNew(c net.Conn) {
	if this.gw != nil && !Options.DisableMetrics {
		this.gw.svrMetrics.ConcurrentPub.Inc(1)
	}
}

func (this *pubServer) onConnClose(c net.Conn) {
	if this.gw != nil && !Options.DisableMetrics {
		this.gw.svrMetrics.ConcurrentPub.Dec(1)
	}
}

func (this *pubServer) respond4XX(appid string, w http.ResponseWriter, err string, status int) {
	if Options.BadPubAppRateLimit && appid != "" && !this.throttleBadAppid.Pour(appid, 1) {
		writeQuotaExceeded(w)
		return
	}

	punishClient()
	w.Header().Set("Connection", "close")
	_writeErrorResponse(w, err, status)
}

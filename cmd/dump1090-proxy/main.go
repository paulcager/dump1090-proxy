package main

import (
	"bufio"
	"encoding/hex"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"io"
	"net"
	"net/http"
	"os"
	"time"

	"dump1090-proxy/beast"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

var (
	listenAddress          = kingpin.Flag("listen-address", "Listen address").Default("localhost:30005").String()
	remotes                = kingpin.Flag("remote", "Remote server(s) to connect to").Required().TCPList()
	dumpMessages           = kingpin.Flag("dumpMessages", "Hex-dump all messages").Bool()
	webListenAddress       = kingpin.Flag("web.listen-address", "Address on which to expose metrics.").Default(":9798").String()
	metricsEndpoint        = kingpin.Flag("web.telemetry-path", "Path under which to expose metrics.").Default("/metrics").String()
	disableExporterMetrics = kingpin.Flag(
		"web.disable-exporter-metrics",
		"TODO - not implemented. Exclude standard runtime metrics (promhttp_*, process_*, go_*).",
	).Bool()
)

// Metrics
var (
	messagesRead = promauto.NewCounter(prometheus.CounterOpts{
		Name: "messages_read",
		Help: "The total number of dump1090 messages read from source",
	})
	messagesWritten = promauto.NewCounter(prometheus.CounterOpts{
		Name: "messages_written",
		Help: "The total number of dump1090 messages written to clients",
	})
	inboundConnections = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "inbound_connections",
		Help: "Number of inbound connections",
	})
	outboundConnections = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "outbound_connections",
		Help: "Number of outbound connections",
	})
	ioErrorCounter = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ioerrors_total",
			Help: `Total IO errors`,
		},
		[]string{"op"},
	)
)

func main() {
	kingpin.Version("dev")
	kingpin.HelpFlag.Short('h')
	kingpin.CommandLine.UsageWriter(os.Stdout)
	kingpin.Parse()

	logger := log.NewLogfmtLogger(os.Stderr)

	listener, err := net.Listen("tcp", *listenAddress)
	if err != nil {
		panic(err)
	}

	go metricServer()
	runProxy(logger, []*net.TCPListener{listener.(*net.TCPListener)}, *remotes)
}

func metricServer() {
	http.Handle(*metricsEndpoint, promhttp.Handler())
	err := http.ListenAndServe(*webListenAddress, nil)
	if err != nil {
		panic(err)
	}
}

func runProxy(logger log.Logger, listeners []*net.TCPListener, remotes []*net.TCPAddr) {
	newConnection := make(chan *net.TCPConn, 4)
	for _, l := range listeners {
		go runListener(logger, l, newConnection)
	}

	newMessage := make(chan []byte, 16)
	for _, r := range remotes {
		go runRemote(logger, r, newMessage)
	}

	clients := make(map[*net.TCPConn]struct{})

	for {
		select {
		case c := <-newConnection:
			level.Info(logger).Log("new_conn", c.RemoteAddr())
			clients[c] = struct{}{}
			c.CloseRead()
			c.SetKeepAlive(true)
			c.SetKeepAlivePeriod(time.Minute)
		case m := <-newMessage:
			if *dumpMessages {
				level.Debug(logger).Log("message", hex.EncodeToString(m))
			}

			for c := range clients {
				c.SetWriteDeadline(time.Now().Add(2 * time.Second))
				_, err := c.Write(m)
				if err != nil {
					ioError(logger, c.RemoteAddr(), "write", err)
					delete(clients, c)
					c.Close()
					continue
				}
			}

			messagesWritten.Inc()
		}

		inboundConnections.Set(float64(len(clients)))
	}
}

func runListener(logger log.Logger, l *net.TCPListener, ch chan<- *net.TCPConn) {
	defer l.Close()
	for {
		conn, err := l.AcceptTCP()
		if err != nil {
			level.Error(logger).Log("listener", l.Addr(), "err", err)
			time.Sleep(time.Second)
			continue
		}

		ch <- conn
	}
}

func runRemote(logger log.Logger, addr *net.TCPAddr, ch chan<- []byte) {
	backoff := time.Duration(0)
	lastErrorLog := time.Time{}

	for {
		time.Sleep(backoff)

		if time.Now().After(lastErrorLog.Add(time.Hour)) {
			level.Info(logger).Log("addr", addr, "action", "connecting")
		}

		conn, err := net.DialTCP("tcp", nil, addr)
		if err != nil {
			if time.Now().After(lastErrorLog.Add(time.Hour)) {
				level.Error(logger).Log("addr", addr, "err", err)
				lastErrorLog = time.Now()
			}

			backoff = (time.Second + backoff) * 2
			if backoff > time.Minute {
				backoff = time.Minute
			}

			continue
		}

		level.Info(logger).Log("addr", addr, "action", "connected")
		backoff = time.Duration(0)
		runRemoteConnection(logger, conn, ch)
	}
}

func runRemoteConnection(logger log.Logger, conn *net.TCPConn, ch chan<- []byte) {

	defer conn.Close()
	defer level.Warn(logger).Log("addr", conn.RemoteAddr().String(), "action", "disconnected")

	outboundConnections.Inc()
	defer outboundConnections.Dec()

	conn.CloseWrite()
	conn.SetKeepAlive(true)
	conn.SetKeepAlivePeriod(time.Minute)

	var r io.Reader = conn
	if *dumpMessages {
		r = NewLoggingReader(r, os.Stderr)
	}

	br := bufio.NewReader(r)

	seenFirstMessage := false
	for {
		b, err := beast.ReadMessage(br)
		if err, ok := err.(beast.InvalidMessage); ok {
			// Don't log warning if we have just connected - may get partial messages.
			if seenFirstMessage {
				level.Warn(logger).Log("err", err)
			}

			// ReadMessage will have consumed at least one byte, so try again with remaining buffer
			continue
		}

		if err != nil {
			ioError(logger, conn.RemoteAddr(), "read", err)
			break
		}

		if !seenFirstMessage {
			level.Info(logger).Log("addr", conn.RemoteAddr().String(), "action", "seenFirstMessage")
		}

		seenFirstMessage = true
		messagesRead.Inc()
		ch <- b
	}
}

func ioError(logger log.Logger, addr interface{}, op string, err error) {
	level.Error(logger).Log("addr", addr, "op", op, "err", err)
	ioErrorCounter.With(prometheus.Labels{
		"op": op,
	}).Inc()
}

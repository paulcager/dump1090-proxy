package main

import (
	"fmt"
	"math"
	"net"
	"os"
	"time"

	"dump1090-proxy/sbs"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	address                = kingpin.Flag("remote", "Dump1090 server to connect to (beast format)").Required().TCP()
	webListenAddress       = kingpin.Flag("web.listen-address", "Address on which to expose metrics.").Default(":9796").String()
	metricsEndpoint        = kingpin.Flag("web.telemetry-path", "Path under which to expose metrics.").Default("/metrics").String()
	disableExporterMetrics = kingpin.Flag(
		"web.disable-exporter-metrics",
		"TODO - not implemented. Exclude standard runtime metrics (promhttp_*, process_*, go_*).",
	).Bool()

	logger log.Logger

	writers = []Writer{
		// We just want the flat file for now.
		// &DbWriter{},
		&FileWriter{},
	}
)

func main() {
	kingpin.Version("dev")
	kingpin.HelpFlag.Short('h')
	kingpin.CommandLine.UsageWriter(os.Stdout)
	kingpin.Parse()

	logger = log.NewLogfmtLogger(os.Stderr)

	ch := make(chan sbs.Message, 32)

	go writer(ch)
	consume(*address, ch)
}

func consume(addr *net.TCPAddr, ch chan sbs.Message) {
	backoff := time.Duration(0)
	lastErrorLog := time.Time{}

	for {
		time.Sleep(backoff)

		if time.Now().After(lastErrorLog.Add(time.Hour)) {
			level.Info(logger).Log("addr", *address, "action", "connecting")
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
		runConnection(conn, ch)
	}
}

func runConnection(conn *net.TCPConn, ch chan<- sbs.Message) {
	defer conn.Close()
	defer level.Warn(logger).Log("addr", conn.RemoteAddr().String(), "action", "disconnected")

	conn.CloseWrite()
	conn.SetKeepAlive(true)
	conn.SetKeepAlivePeriod(time.Minute)
	reader := sbs.NewReader(conn)

	seenFirstMessage := false
	for {
		m, err := reader.Read()

		if err != nil {
			level.Error(logger).Log("err", err)
			break
		}

		if !seenFirstMessage {
			level.Info(logger).Log("addr", conn.RemoteAddr().String(), "action", "seenFirstMessage")
		}

		seenFirstMessage = true

		ch <- m
	}
}

var (
	nextRotate time.Time
)

func writer(ch <-chan sbs.Message) {
	count := 0

	defer func() {
		for _, w := range writers {
			w.Close()
		}
	}()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			for _, w := range writers {
				w.Flush()
			}

		case m := <-ch:
			if math.IsNaN(m.Latitude) || math.IsNaN(m.Longitude) {
				continue
			}

			count++
			if count%1000 == 0 {
				fmt.Println(count, int(m.Type), m.Timestamp, m.HexIdent, m.Latitude, m.Longitude, m.Altitude)
			}

			if nextRotate.IsZero() || m.Timestamp.After(nextRotate) {
				// Need to rotate files
				for _, w := range writers {
					if err := w.Rotate(m.Timestamp); err != nil {
						panic(err)
					}
				}
				nextRotate = m.Timestamp.Truncate(24 * time.Hour).Add(24 * time.Hour)
			}

			for _, w := range writers {
				err := w.Write(m)
				if err != nil {
					// Almost certainly unrecoverable.
					panic(err)
				}
			}
		}
	}
}

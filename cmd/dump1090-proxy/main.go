package main

import (
	"bufio"
	"dump1090-proxy/beast"
	"encoding/hex"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
	"net"
	"os"
	"time"
)

var (
	listenAddress = kingpin.Flag("listen-address", "Listen address").Default("localhost:30005").String()
	remotes       = kingpin.Flag("remote", "Remote server(s) to connect to").Required().TCPList()
	dumpMessages  = kingpin.Flag("dumpMessages", "Hex-dump all messages").Bool()
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

	runProxy(logger, []*net.TCPListener{listener.(*net.TCPListener)}, *remotes)
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
			level.Info(logger).Log("New conn", c.RemoteAddr())
			clients[c] = struct{}{}
			c.CloseRead()
			c.SetKeepAlive(true)
			c.SetKeepAlivePeriod(time.Minute)
		case m := <-newMessage:
			if *dumpMessages {
				level.Debug(logger).Log("message", hex.EncodeToString(m))
			}
			for c := range clients {
				c.SetDeadline(time.Now().Add(2 * time.Second))
				_, err := c.Write(m)
				if err != nil {
					ioError(logger, c.RemoteAddr(), "write", err)
					delete(clients, c)
					c.Close()
				}
			}
		}
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
	for {
		time.Sleep(backoff)
		conn, err := net.DialTCP("tcp", nil, addr)
		if err != nil {
			level.Error(logger).Log("addr", addr, "err", err)
			backoff = (time.Second + backoff) * 2
			if backoff > time.Minute {
				backoff = time.Minute
			}

			continue
		}

		runConnection(logger, conn, ch)
	}
}

func runConnection(logger log.Logger, conn *net.TCPConn, ch chan<- []byte) {

	defer conn.Close()

	conn.CloseWrite()

	r := bufio.NewReader(conn)
	seenFirstMessage := false
	for {
		b, err := beast.ReadMessage(r)
		if err == beast.InvalidMessage {
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

		seenFirstMessage = true

		ch <- b
	}
}

func ioError(logger log.Logger, addr interface{}, op string, err error) {
	level.Error(logger).Log("addr", addr, "op", op, "err", err)
}

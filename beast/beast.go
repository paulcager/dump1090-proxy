// Package beast decodes the ADSB "beast" representation defined in
// https://wiki.jetvision.de/wiki/Mode-S_Beast:Data_Output_Formats
//
// There are only three frame formats in the binary protocol, in order to keep some control characters, the Mode-S Beast uses a so called escaped binary format:
//
//<esc> "1" : 6 byte MLAT timestamp, 1 byte signal level, 2 byte Mode-AC
//<esc> "2" : 6 byte MLAT timestamp, 1 byte signal level, 7 byte Mode-S short frame
//<esc> "3" : 6 byte MLAT timestamp, 1 byte signal level, 14 byte Mode-S long frame
//<esc><esc>: true 0x1a
//
//<esc> is 0x1a, and "1", "2" and "3" are 0x31, 0x32 and 0x33
package beast

import (
	"bufio"
	"encoding/hex"
	"errors"
	"fmt"
)

const (
	esc = 0x1a
)

//type Message struct {
//	Timestamp   time.Time
//	SignalLevel byte
//	Data        []byte
//}

var (
	InvalidMessage = errors.New("unexpected start of message")
)

func ReadMessage(r *bufio.Reader) ([]byte, error) {
	// We can't just forward bytes from all connections since we might
	// send half a message from one connection followed by half from a
	// different connection: we need to treat whole messages atomically.
	//
	// This causes problems because messages *start* with an escape byte,
	// but we don't want to introduce latency by waiting for the next
	// message's escape byte. Fortunately, the messages we are interested
	// in all have a known length - but we can't just read that number of
	// bytes because of escape characters which are themselves escaped.

	// The maximum sized message we are interested in
	const maxSize = 1 + 6 + 1 + 14
	var (
		payloadLength int
	)

	// The first byte should be a 0x1a, followed by the message type.
	start, err := r.ReadByte()
	if err != nil {
		return nil, err
	}
	if start != esc {
		return nil, InvalidMessage
	}

	mType, err := r.ReadByte()
	if err != nil {
		return nil, err
	}

	if mType == esc {
		// Not a valid start of message
		return nil, InvalidMessage
	}

	switch mType {
	case '1':
		payloadLength = 2
	case '2':
		payloadLength = 7
	case '3':
		payloadLength = 14
	default:
		// Something weird going on, maybe a new message type whose length we don't know.
		return nil, nil
	}

	// 2-byte header, 6-byte timestamp, 1 byte signal level + payload
	buff := make([]byte, 2+6+1+payloadLength)
	buff[0] = esc
	buff[1] = mType
	i := 2
	for i < len(buff) {
		b, err := r.ReadByte()
		if err != nil {
			return nil, err
		}

		buff[i] = b
		i++

		if b == esc {
			b, err := r.ReadByte()
			if err != nil {
				return nil, err
			}
			if b != esc {
				return nil, fmt.Errorf("unescaped escape %c after %s", b, hex.Dump(buff[:i]))
			}
		}
	}

	//fmt.Println(hex.Dump(buff))
	return buff, err
}

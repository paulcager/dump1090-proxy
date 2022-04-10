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
	"io"
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

	// 2-byte header, 6-byte timestamp, 1 byte signal level + payload
	// The payload can be up to twice its declared length, because escape characters may
	// need escaping.
	const fixedSize = 2 + 6 + 1
	const maxSize = fixedSize + 2*14

	// Read header.
	buff := make([]byte, maxSize)
	_, err := io.ReadFull(r, buff[:fixedSize])
	if err != nil {
		return nil, err
	}

	// The first byte should be a 0x1a, followed by the message type.
	if buff[0] != esc || buff[1] == esc {
		return nil, InvalidMessage
	}

	var (
		mType         = buff[1]
		payloadLength int
	)

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

	pos := fixedSize
	for i := 0; i < payloadLength; i++ {
		b, err := r.ReadByte()
		if err != nil {
			return nil, err
		}

		buff[pos] = b
		pos++

		if b == esc {
			// Sould be followed by an escape
			b, err := r.ReadByte()
			if err != nil {
				return nil, err
			}
			if b != esc {
				return nil, fmt.Errorf("unescaped escape %c after %s", b, hex.Dump(buff[:i]))
			}
			buff[pos] = b
			pos++
		}
	}

	return buff[:pos], err
}

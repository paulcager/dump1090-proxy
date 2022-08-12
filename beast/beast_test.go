package beast

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNoData(t *testing.T) {
	s := str("^311111111234567890123")
	_, err := ReadMessage(bufio.NewReader(bytes.NewReader(s)))
	assert.Error(t, err, io.ErrUnexpectedEOF)
}

func TestTwoMessages(t *testing.T) {
	s := str("^3111111112345678901234^3111111177777777777777")
	assert.Equal(t, 2*(9+14), len(s))
	r := bufio.NewReader(bytes.NewReader(s))
	b, err := ReadMessage(r)
	noError(t, err)
	assert.Equal(t, 9+14, len(b))
	assert.Equal(t, str("^3111111112345678901234"), b)

	b, err = ReadMessage(r)
	noError(t, err)
	assert.Equal(t, 9+14, len(b))
	assert.Equal(t, str("^3111111177777777777777"), b)
}

func TestEscapes(t *testing.T) {
	s := str("^311111111234^^678901234^311111117777777777777^^")
	assert.Equal(t, (9+15)+(9+15), len(s))
	r := bufio.NewReader(bytes.NewReader(s))

	b, err := ReadMessage(r)
	noError(t, err)
	assert.Equal(t, 9+15, len(b))
	assert.Equal(t, str("^311111111234^^678901234"), b)

	b, err = ReadMessage(r)
	noError(t, err)
	assert.Equal(t, 9+15, len(b))
	assert.Equal(t, str("^311111117777777777777^^"), b)
}

func TestTimestampContainsEsc(t *testing.T) {
	b, err := hex.DecodeString(
		"" +
			"1a32095545b3dda7275da07dc82b29ed" +
			"1a33095545b41a1a697c8d406e69990cd82c1808026c12ab" +
			"1a32095545b4d90e655d4008f29bcbab")
	noError(t, err)
	r := bufio.NewReader(bytes.NewReader(b))
	var msgs [][]byte
	for {
		msg, err := ReadMessage(r)
		if err == io.EOF {
			break
		}
		noError(t, err)
		msgs = append(msgs, msg)
	}

	assert.Equal(t, 3, len(msgs))
}

func str(s string) []byte {
	return []byte(strings.ReplaceAll(s, "^", "\x1a"))
}

func noError(t *testing.T, err error) {
	if err != nil {
		t.Error(err)
	}
}

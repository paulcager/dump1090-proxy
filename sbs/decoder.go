package sbs

import (
	"encoding/csv"
	"io"
	"math"
	"strconv"
	"time"
)

const (
	transmissionMessage = "MSG"
	expectedFields      = 22
)

//go:generate stringer -type=MessageType
type MessageType int

const (
	IdAndCategory MessageType = iota + 1
	SurfacePosition
	AirbornePosition
	AirborneVelocity
	Alt
	ID
	AitToAir
	AllCallReply
)

type Message struct {
	Type         MessageType
	HexIdent     string
	Timestamp    time.Time
	Callsign     string
	Altitude     float64
	GroundSpeed  float64
	Track        float64
	Latitude     float64
	Longitude    float64
	VerticalRate float64
	Squark       string
	Alert        bool
	Emergency    bool
	Ident        bool
	OnGound      bool
}

type Reader struct {
	csvReader *csv.Reader
}

func (r *Reader) Read() (Message, error) {
	for {
		record, err := r.csvReader.Read()
		if err != nil {
			return Message{}, err
		}

		if len(record) < expectedFields {
			continue
		}

		if record[0] != transmissionMessage {
			// We are not interested in these - they relate to the BaseStation hardware
			// itself.

			continue
		}

		transType, err := strconv.ParseUint(record[1], 10, 8)
		if err != nil || transType > uint64(AllCallReply) {
			continue
		}

		return Message{
			Type:         MessageType(transType),
			HexIdent:     record[4],
			Timestamp:    timeOf(record[6], record[7]),
			Callsign:     record[10],
			Altitude:     number(record[11]),
			GroundSpeed:  number(record[12]),
			Track:        number(record[13]),
			Latitude:     number(record[14]),
			Longitude:    number(record[15]),
			VerticalRate: number(record[16]),
			Squark:       record[17],
			Alert:        boolean(record[18]),
			Emergency:    boolean(record[19]),
			Ident:        boolean(record[20]),
			OnGound:      boolean(record[21]),
		}, nil
	}
}

func number(s string) float64 {
	if s == "" {
		return math.NaN()
	}

	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return f
	}

	return math.NaN()
}

var (
	timeFormat = "2006/01/02 15:04:05.000"
)

func timeOf(d string, t string) time.Time {
	if d == "" || t == "" {
		return time.Time{}
	}

	timestamp, err := time.Parse(timeFormat, d+" "+t)

	if err != nil {
		return time.Time{}
	}

	return timestamp
}

func boolean(s string) bool {
	return s == "-1"
}

func NewReader(r io.Reader) *Reader {
	return &Reader{
		csvReader: csv.NewReader(r),
	}
}

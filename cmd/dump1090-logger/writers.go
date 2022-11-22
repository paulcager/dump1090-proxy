package main

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"os"
	"time"

	"dump1090-proxy/sbs"
	"github.com/go-kit/log/level"
	_ "github.com/mattn/go-sqlite3"
)

type Writer interface {
	Write(m sbs.Message) error
	Rotate(timestamp time.Time) error
	Close() error
}

type DbWriter struct {
	db   *sql.DB
	stmt *sql.Stmt
}

func (db *DbWriter) Write(m sbs.Message) error {
	_, err := db.stmt.Exec(int(m.Type), m.Timestamp, m.HexIdent, m.Latitude, m.Longitude, m.Altitude)
	return err
}

func (db *DbWriter) Rotate(timestamp time.Time) error {
	_ = db.Close()

	fileName := "./dump1090-" + timestamp.Format("2006-01-02") + ".db"
	level.Info(logger).Log("rotating", fileName)

	var err error
	db.db, err = sql.Open("sqlite3", fileName)
	if err != nil {
		return err
	}

	// Ignore failures ("already exists")
	_, _ = db.db.Exec(`
			create table messages(type integer, timestamp text, hexIdent text, lat float64, lon float64, alt float64)
		`)

	if err != nil {
		return err
	}

	db.stmt, err = db.db.Prepare(`
			insert into messages(type, timestamp , hexIdent, lat, lon, alt)
			values(?, strftime('%Y-%m-%d %H:%M:%f', ?), ?, ?, ?, ?)`)

	return err
}

func (db *DbWriter) Close() error {
	if db.stmt != nil {
		db.stmt.Close()
	}
	if db.db != nil {
		return db.db.Close()
	}

	return nil
}

type FileWriter struct {
	file *os.File
	c    *csv.Writer
}

func (db *FileWriter) Write(m sbs.Message) error {
	err := db.c.Write([]string{
		fmt.Sprint(int(m.Type)),
		m.Timestamp.Format("2006-01-02T15:04:05"),
		m.HexIdent,
		fmt.Sprint(m.Latitude),
		fmt.Sprint(m.Longitude),
		fmt.Sprint(m.Altitude),
	})
	return err
}

func (db *FileWriter) Rotate(timestamp time.Time) error {
	_ = db.Close()

	fileName := "./dump1090-" + timestamp.Format("2006-01-02") + ".csv"
	level.Info(logger).Log("rotating", fileName)

	var err error
	db.file, err = os.OpenFile(fileName, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return err
	}

	db.c = csv.NewWriter(db.file)

	return nil
}

func (db *FileWriter) Close() error {
	if db.c != nil {
		db.c.Flush()
	}

	if db.file != nil {
		return db.file.Close()
	}

	return nil
}

var (
	_ Writer = &DbWriter{}
	_ Writer = &FileWriter{}
)

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
	Flush()
	Close() error
}

type DbWriter struct {
	db   *sql.DB
	stmt *sql.Stmt
}

func (db *DbWriter) Flush() {
	// No action needed
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

func (fw *FileWriter) Flush() {
	fw.c.Flush()
}

func (fw *FileWriter) Write(m sbs.Message) error {
	err := fw.c.Write([]string{
		fmt.Sprint(int(m.Type)),
		m.Timestamp.Format("2006-01-02T15:04:05.000"),
		m.HexIdent,
		fmt.Sprint(m.Latitude),
		fmt.Sprint(m.Longitude),
		fmt.Sprint(m.Altitude),
	})
	return err
}

func (fw *FileWriter) Rotate(timestamp time.Time) error {
	_ = fw.Close()

	fileName := "./dump1090-" + timestamp.Format("2006-01-02") + ".csv"
	level.Info(logger).Log("rotating", fileName)

	var err error
	fw.file, err = os.OpenFile(fileName, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return err
	}

	fw.c = csv.NewWriter(fw.file)

	return nil
}

func (fw *FileWriter) Close() error {
	if fw.c != nil {
		fw.c.Flush()
	}

	if fw.file != nil {
		return fw.file.Close()
	}

	return nil
}

var (
	_ Writer = &DbWriter{}
	_ Writer = &FileWriter{}
)

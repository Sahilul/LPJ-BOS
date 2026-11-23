package app

import (
	"os"
	"path/filepath"
	"strings"
	"time"
	_ "time/tzdata"
)

type Config struct {
	DataDir       string
	DBPath        string
	UploadDir     string
	TmpDir        string
	MaxImageBytes int64
}

var Cfg Config

// startedAt dipakai sebagai versi aset (?v=) karena aset tertanam di binary.
var startedAt = time.Now().Unix()

func loadConfig() Config {
	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		wd, _ := os.Getwd()
		dataDir = filepath.Join(wd, "data")
		// Saat dijalankan dari folder src/, data ada di root repo.
		if filepath.Base(wd) == "src" {
			dataDir = filepath.Join(filepath.Dir(wd), "data")
		}
	}
	dataDir = strings.TrimRight(dataDir, "/")
	c := Config{
		DataDir:       dataDir,
		DBPath:        os.Getenv("DB_PATH"),
		UploadDir:     dataDir + "/uploads",
		TmpDir:        dataDir + "/tmp",
		MaxImageBytes: 2 * 1024 * 1024,
	}
	if c.DBPath == "" {
		c.DBPath = dataDir + "/lpjbos.sqlite"
	}
	return c
}

func setupTimezone() {
	tz := os.Getenv("TZ")
	if tz == "" {
		tz = "Asia/Jakarta"
	}
	if loc, err := time.LoadLocation(tz); err == nil {
		time.Local = loc
	}
}

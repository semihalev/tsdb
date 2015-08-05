package main

import (
	"bytes"
	"flag"
	"fmt"
	"github.com/boltdb/bolt"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"time"
)

type Data struct {
	Time  string      `json:"time"`
	Value interface{} `json:"value"`
}

var (
	flaghttp     = flag.String("http", ":4080", "web server listen addr")
	flagcpus     = flag.Int("cpus", 1, "Set the maximum number of CPUs to use")
	flagdb       = flag.String("db", "ts.db", "Database path")
	flagsynctime = flag.String("sync", "1s", "sync time in seconds, if nosync=true")
	flagnosync   = flag.Bool("nosync", false, "auto sync")
	flagexpire   = flag.String("expire", "0", "default key expire time  default:no-expire")
	flagpprof    = flag.Bool("pprof", false, "enable disable Go profiling")
)

var db *bolt.DB
var stats bolt.Stats
var defaultexpire time.Duration

const (
	TTL_SERIES_META = "_TTL_SERIES"
)

func query(c *gin.Context) {
	series := c.Query("series")
	if series == "" {
		c.JSON(http.StatusOK, gin.H{
			"status":  "error",
			"message": "series need",
		})
		return
	}

	limit := c.Query("limit")
	if limit == "" {
		limit = "0"
	}

	offset := c.Query("offset")
	if offset == "" {
		offset = "0"
	}

	order := c.Query("order")
	if order == "" {
		order = "desc"
	}

	var m []*Data

	err := db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(series))
		if bucket == nil {
			return fmt.Errorf("Series not found!")
		}

		cursor := bucket.Cursor()

		l, _ := strconv.Atoi(limit)
		o, _ := strconv.Atoi(offset)

		startfunc := cursor.Last
		nextfunc := cursor.Prev

		if order == "asc" {
			startfunc = cursor.First
			nextfunc = cursor.Next
		}

		for k, v := startfunc(); k != nil; k, v = nextfunc() {
			if o > 0 {
				o--
				continue
			}

			m = append(m, &Data{Time: string(k), Value: string(v)})
			//m[string(k)] = string(v)

			if l > 0 {
				l--
				if l == 0 {
					break
				}
			}
		}

		return nil
	})

	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"status":  "error",
			"message": err.Error(),
		})

		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
		"result": m,
	})
}

func asyncwrite(c *gin.Context) {
	series := c.Query("series")
	if series == "" {
		c.JSON(http.StatusOK, gin.H{
			"status":  "error",
			"message": "series need",
		})
		return
	}

	keytime := c.Query("time")
	if keytime == "" {
		keytime = strconv.FormatInt(time.Now().UnixNano(), 10)
	}

	value := c.Query("value")
	if value == "" {
		c.JSON(http.StatusOK, gin.H{
			"status":  "error",
			"message": "value need",
		})
		return
	}

	ttl := c.Query("ttl")
	if ttl != "" {
		_, err := time.ParseDuration(ttl)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"status":  "error",
				"message": err.Error(),
			})
			return
		}
	} else {
		ttl = "0"
	}

	go func() {
		err := db.Batch(func(tx *bolt.Tx) error {
			bucket := tx.Bucket([]byte(series))
			var err error
			if bucket == nil {
				bucket, err = tx.CreateBucket([]byte(series))
				if err != nil {
					return err
				}

				bttl := tx.Bucket([]byte(TTL_SERIES_META))
				bttl.Put([]byte(series), []byte(ttl))
			}

			key := []byte(keytime)

			if err != nil {
				return err
			}

			err = bucket.Put(key, []byte(value))
			if err != nil {
				return err
			}

			return nil
		})

		if err != nil {
			log.Println("async write error", err)
		}
	}()

	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
	})
}

func write(c *gin.Context) {
	series := c.Query("series")
	if series == "" {
		c.JSON(http.StatusOK, gin.H{
			"status":  "error",
			"message": "series need",
		})
		return
	}

	keytime := c.Query("time")
	if keytime == "" {
		keytime = strconv.FormatInt(time.Now().UnixNano(), 10)
	}

	value := c.Query("value")
	if value == "" {
		c.JSON(http.StatusOK, gin.H{
			"status":  "error",
			"message": "value need",
		})
		return
	}

	ttl := c.Query("ttl")
	if ttl != "" {
		_, err := time.ParseDuration(ttl)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"status":  "error",
				"message": err.Error(),
			})
			return
		}
	} else {
		ttl = "0"
	}

	err := db.Batch(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(series))
		var err error
		if bucket == nil {
			bucket, err = tx.CreateBucket([]byte(series))
			if err != nil {
				return err
			}

			bttl := tx.Bucket([]byte(TTL_SERIES_META))
			bttl.Put([]byte(series), []byte(ttl))
		}

		key := []byte(keytime)

		if err != nil {
			return err
		}

		err = bucket.Put(key, []byte(value))
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"status":  "error",
			"message": err.Error(),
		})

		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
	})
}

func delete(c *gin.Context) {
	series := c.Query("series")
	if series == "" {
		c.JSON(http.StatusOK, gin.H{
			"status":  "error",
			"message": "series need",
		})
		return
	}

	err := db.Batch(func(tx *bolt.Tx) error {
		err := tx.DeleteBucket([]byte(series))
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"status":  "error",
			"message": err.Error(),
		})

		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
	})
}

func count(c *gin.Context) {
	series := c.Query("series")
	if series == "" {
		c.JSON(http.StatusOK, gin.H{
			"status":  "error",
			"message": "series need",
		})
		return
	}

	var cnt int

	err := db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(series))
		if bucket == nil {
			return fmt.Errorf("Series not found!")
		}

		stats := bucket.Stats()
		cnt = stats.KeyN

		return nil
	})

	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
			"result": 0,
		})

		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
		"result": cnt,
	})
}

func deletebytime(c *gin.Context) {
	series := c.Query("series")
	if series == "" {
		c.JSON(http.StatusOK, gin.H{
			"status":  "error",
			"message": "series need",
		})
		return
	}

	key := c.Query("time")
	if key == "" {
		c.JSON(http.StatusOK, gin.H{
			"status":  "error",
			"message": "time need",
		})
		return
	}

	err := db.Batch(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(series))
		if bucket == nil {
			return fmt.Errorf("Series not found!")
		}
		return bucket.Delete([]byte(key))
	})

	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"status":  "error",
			"message": err.Error(),
		})

		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
	})
}

func dbstats(c *gin.Context) {
	c.JSON(http.StatusOK, stats)
}

func backup(c *gin.Context) {
	if *flagnosync {
		db.Sync()
	}

	err := db.View(func(tx *bolt.Tx) error {
		filename := fmt.Sprintf("backup_%d.db", time.Now().UnixNano())

		c.Writer.Header().Set("Content-Type", "application/octet-stream")
		c.Writer.Header().Set("Content-Disposition", "attachment; filename="+filename)
		c.Writer.Header().Set("Content-Length", strconv.Itoa(int(tx.Size())))
		_, err := tx.WriteTo(c.Writer)

		return err
	})

	if err != nil {
		c.String(http.StatusInternalServerError, "database backup error")
	}
}

func expireKeys(series []byte, keys [][]byte) {
	db.Batch(func(tx *bolt.Tx) error {
		b := tx.Bucket(series)
		if b == nil {
			return nil
		}

		for _, key := range keys {
			b.Delete(key)
		}

		return nil
	})
}

func expire() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for _ = range ticker.C {
		log.Println("Expire keys batch running...")

		expirecounter := 0

		db.View(func(tx *bolt.Tx) error {
			now := time.Now()
			bttl := tx.Bucket([]byte(TTL_SERIES_META))
			var expiretime string

			tx.ForEach(func(series []byte, b *bolt.Bucket) error {
				if string(series) == TTL_SERIES_META {
					return nil
				}

				ttl, _ := time.ParseDuration(string(bttl.Get(series)))

				if ttl > 0 {
					expiretime = strconv.FormatInt(now.Add(-ttl).UnixNano(), 10)
				} else if defaultexpire > 0 {
					expiretime = strconv.FormatInt(now.Add(-defaultexpire).UnixNano(), 10)
				} else {
					return nil
				}

				c := b.Cursor()

				min := []byte("1")
				max := []byte(expiretime)

				var keys [][]byte
				for k, _ := c.Seek(min); k != nil && bytes.Compare(k, max) <= 0; k, _ = c.Next() {
					keys = append(keys, k)
					expirecounter++
				}

				if len(keys) > 0 {
					go expireKeys(series, keys)
				}

				return nil
			})

			return nil
		})

		log.Println("Expire keys batch finished.", expirecounter, "key expired.")
	}
}

func main() {
	flag.Parse()

	if *flagcpus == 0 {
		runtime.GOMAXPROCS(runtime.NumCPU())
	} else {
		runtime.GOMAXPROCS(*flagcpus)
	}

	var err error
	db, err = bolt.Open(*flagdb, 0644, nil)
	if err != nil {
		log.Fatal("db can't open", err)
	}
	defer db.Close()

	db.Update(func(tx *bolt.Tx) error {
		tx.CreateBucketIfNotExists([]byte(TTL_SERIES_META))
		return nil
	})

	db.NoSync = *flagnosync

	defaultexpire, err = time.ParseDuration(*flagexpire)
	if err != nil {
		log.Fatal("parse expire duration error", err)
	}

	sync, err := time.ParseDuration(*flagsynctime)
	if err != nil {
		log.Fatal("parse sync duration error", err)
	}

	go expire()

	go func() {
		prev := db.Stats()

		for {
			tmpstats := db.Stats()
			stats = tmpstats.Sub(&prev)

			prev = tmpstats

			time.Sleep(10 * time.Second)
		}
	}()

	if *flagnosync {
		go func() {
			for {
				db.Sync()

				time.Sleep(sync)
			}
		}()
	}

	r := gin.Default()

	gin.SetMode(gin.ReleaseMode)

	v1 := r.Group("/api/v1")
	{
		v1.GET("/stats", dbstats)
		v1.GET("/query", query)
		v1.GET("/delete", delete)
		v1.GET("/write", write)
		v1.GET("/asyncwrite", asyncwrite)
		v1.GET("/count", count)
		v1.GET("/deletebytime", deletebytime)
		v1.GET("/backup", backup)
	}

	PProf(r)

	go func() {
		srv := &http.Server{
			Addr:         *flaghttp,
			Handler:      r,
			ReadTimeout:  60 * time.Second,
			WriteTimeout: 60 * time.Second,
		}

		if err := srv.ListenAndServe(); err != nil {
			log.Fatal(err)
		}
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)

	<-c

	if *flagnosync {
		db.Sync()
	}
}

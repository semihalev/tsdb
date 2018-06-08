package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	log "github.com/inconshreveable/log15"
	stats "github.com/semihalev/gin-stats"
	"github.com/tidwall/buntdb"
)

var (
	flaghttp   = flag.String("http", ":4080", "http server addr")
	flagcpus   = flag.Int("C", 1, "set the maximum number of CPUs to use")
	flagdb     = flag.String("db", "ts.db", "database path")
	flagexpire = flag.Duration("expire", time.Duration(0), "default data expire period")
	flagLogLvl = flag.String("L", "info", "Log verbosity level [crit,error,warn,info,debug]")
)

const (
	version = "2.0-rc1"
)

var db *buntdb.DB

func query(c *gin.Context) {
	series := c.Query("series")
	if series == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "series name required",
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

	type data struct {
		Time  int64       `json:"time"`
		Value interface{} `json:"value"`
	}

	var m []*data

	err := db.View(func(tx *buntdb.Tx) error {
		l, _ := strconv.Atoi(limit)
		o, _ := strconv.Atoi(offset)

		iterFunc := tx.DescendKeys

		if order == "asc" {
			iterFunc = tx.AscendKeys
		}

		err := iterFunc(series+":*", func(k, v string) bool {
			if o > 0 {
				o--
				return true
			}

			nanoTime, _ := strconv.ParseInt(strings.TrimPrefix(k, series+":"), 10, 64)

			m = append(m, &data{Time: nanoTime, Value: v})

			if l > 0 {
				l--
				if l == 0 {
					return false
				}
			}

			return true
		})

		return err
	})

	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":  "error",
			"message": err.Error(),
		})

		return
	}

	if m == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"status":  "error",
			"message": "no data found",
		})

		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
		"result": m,
	})
}

func write(c *gin.Context) {
	series := c.Query("series")
	if series == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "series name required",
		})
		return
	}

	keytime := c.Query("time")
	if keytime == "" {
		keytime = strconv.FormatInt(time.Now().UnixNano(), 10)
	}

	value := c.Query("value")
	if value == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "value field required",
		})
		return
	}

	var ttl time.Duration
	var err error
	if c.Query("ttl") != "" {
		ttl, err = time.ParseDuration(c.Query("ttl"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"status":  "error",
				"message": err.Error(),
			})
			return
		}
	} else {
		ttl = *flagexpire
	}

	err = db.Update(func(tx *buntdb.Tx) error {
		key := series + ":" + keytime

		opts := &buntdb.SetOptions{Expires: false, TTL: ttl}
		if ttl != 0 {
			opts.Expires = true
		}

		_, _, err := tx.Set(key, value, opts)
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
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
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "series name required",
		})
		return
	}

	err := db.Update(func(tx *buntdb.Tx) error {
		var delkeys []string
		tx.AscendKeys(series+":*", func(k, v string) bool {
			delkeys = append(delkeys, k)
			return true
		})

		for _, k := range delkeys {
			if _, err := tx.Delete(k); err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
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
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "series name required",
		})
		return
	}

	var cnt int

	err := db.View(func(tx *buntdb.Tx) error {
		tx.AscendKeys(series+":*", func(k, v string) bool {
			cnt++
			return true
		})

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
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "series name required",
		})
		return
	}

	key := c.Query("time")
	if key == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "time field required",
		})
		return
	}

	err := db.Update(func(tx *buntdb.Tx) error {
		_, err := tx.Delete(series + ":" + key)

		return err
	})

	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":  "error",
			"message": err.Error(),
		})

		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
	})
}

func backup(c *gin.Context) {
	filename := fmt.Sprintf("backup_%d.db", time.Now().UnixNano())

	c.Writer.Header().Set("Content-Type", "application/octet-stream")
	c.Writer.Header().Set("Content-Disposition", "attachment; filename="+filename)

	err := db.Save(c.Writer)

	if err != nil {
		c.String(http.StatusInternalServerError, "database backup error")
	}
}

func getStats(c *gin.Context) {
	c.JSON(http.StatusOK, stats.Report())
}

func runWebServer() {
	r := gin.Default()

	r.Use(stats.RequestStats())

	v1 := r.Group("/api/v1")
	{
		v1.GET("/query", query)
		v1.GET("/write", write)
		v1.GET("/count", count)
		v1.GET("/delete", delete)
		v1.GET("/deletebytime", deletebytime)
	}

	r.GET("/backup", backup)
	r.GET("/stats", getStats)

	go func() {
		srv := &http.Server{
			Addr:         *flaghttp,
			Handler:      r,
			ReadTimeout:  60 * time.Second,
			WriteTimeout: 60 * time.Second,
		}

		if err := srv.ListenAndServe(); err != nil {
			log.Crit("HTTP server listen fault", "error", err.Error())
			os.Exit(1)
		}
	}()
}

func init() {
	gin.SetMode(gin.ReleaseMode)
}

func main() {
	flag.Parse()

	if *flagcpus == 0 {
		runtime.GOMAXPROCS(runtime.NumCPU())
	} else {
		runtime.GOMAXPROCS(*flagcpus)
	}

	lvl, err := log.LvlFromString(*flagLogLvl)
	if err != nil {
		log.Crit("Log verbosity level unknown")
		os.Exit(1)
	}

	log.Root().SetHandler(log.LvlFilterHandler(lvl, log.StdoutHandler))

	db, err = buntdb.Open(*flagdb)
	if err != nil {
		log.Crit("Database cannot open", "error", err.Error())
		os.Exit(1)
	}

	defer db.Close()

	runWebServer()

	log.Info("TSDB service started", "version", "v"+version)

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGKILL, syscall.SIGTERM)

	<-c

	log.Info("TSDB service stopping")
}

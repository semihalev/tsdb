package main

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/tidwall/buntdb"
)

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	gin.SetMode(gin.TestMode)
	gin.DefaultWriter, _ = os.Open(os.DevNull)

	var transport http.RoundTripper = &http.Transport{
		Dial:                dialTimeout,
		DisableCompression:  true,
		MaxIdleConnsPerHost: 200,
	}

	http.DefaultTransport = transport

	db, _ = buntdb.Open(":memory:")
	runWebServer("127.0.0.1:4080")
}

func dialTimeout(network, addr string) (net.Conn, error) {
	c, err := (&net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}).Dial(network, addr)

	if err != nil {
		return nil, err
	}

	return c, nil
}

func fetch(url string) (string, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != 200 {
		return string(data), fmt.Errorf("status code %d", resp.StatusCode)
	}

	return string(data), nil
}

func writeHelloWorld(t *testing.T) {

	result, err := fetch("http://127.0.0.1:4080/api/v1/write?series=world&value=hello")
	if err != nil {
		t.Errorf("Error: %v", err)
		return
	}
	assert.Contains(t, result, "\"status\":\"ok\"")
}

func deleteHelloWorld(t *testing.T) {
	result, err := fetch("http://127.0.0.1:4080/api/v1/delete?series=world")
	if err != nil {
		t.Errorf("Error: %v", err)
		return
	}
	assert.Contains(t, result, "\"status\":\"ok\"")
}

func TestWrite(t *testing.T) {
	writeHelloWorld(t)
	deleteHelloWorld(t)
}

func TestQuery(t *testing.T) {
	writeHelloWorld(t)
	result, err := fetch("http://127.0.0.1:4080/api/v1/query?series=world")
	if err != nil {
		t.Errorf("Error: %v", err)
		return
	}
	assert.Contains(t, result, "\"status\":\"ok\"")
	deleteHelloWorld(t)
}

func TestDelete(t *testing.T) {
	deleteHelloWorld(t)
}

func TestCount(t *testing.T) {
	writeHelloWorld(t)
	result, err := fetch("http://127.0.0.1:4080/api/v1/count?series=world")
	if err != nil {
		t.Errorf("Error: %v", err)
		return
	}
	assert.Contains(t, result, "\"result\":1")
	deleteHelloWorld(t)
}

func Benchmark_Write(b *testing.B) {
	for n := 0; n < b.N; n++ {
		if _, err := fetch("http://127.0.0.1:4080/api/v1/write?series=world&value=hello"); err != nil {
			b.Fail()
		}
	}
}

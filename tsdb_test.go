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

	time.Sleep(500 * time.Millisecond)
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

func fetch(uri string) (string, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Get(fmt.Sprintf("http://127.0.0.1:4080%s", uri))
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

func writeHelloWorld() (string, error) {
	result, err := fetch("/api/v1/write?series=world&value=hello")
	if err != nil {
		return "", err
	}
	return result, nil
}

func deleteHelloWorld() (string, error) {
	result, err := fetch("/api/v1/delete?series=world")
	if err != nil {
		return "", err
	}
	return result, nil
}

func TestWrite(t *testing.T) {
	result, err := writeHelloWorld()
	assert.NoError(t, err)
	assert.Contains(t, result, "\"status\":\"ok\"")

	_, err = deleteHelloWorld()
	assert.NoError(t, err)
}

func TestWriteFail(t *testing.T) {
	_, err := fetch("/api/v1/write")
	assert.EqualError(t, err, "status code 400")
	_, err = fetch("/api/v1/write?series=world")
	assert.EqualError(t, err, "status code 400")
	_, err = fetch("/api/v1/write?series=world&value=hello&ttl=5")
	assert.EqualError(t, err, "status code 400")
}

func TestWriteAll(t *testing.T) {
	result, err := fetch("/api/v1/write?series=world&value=hellottl&ttl=5m")
	assert.NoError(t, err)
	assert.Contains(t, result, "\"status\":\"ok\"")
}

func TestQuery(t *testing.T) {
	_, err := writeHelloWorld()
	assert.NoError(t, err)
	result, err := fetch("/api/v1/query?series=world")
	assert.NoError(t, err)
	assert.Contains(t, result, "\"status\":\"ok\"")
	_, err = deleteHelloWorld()
	assert.NoError(t, err)
}

func TestQueryAll(t *testing.T) {
	_, err := writeHelloWorld()
	assert.NoError(t, err)
	_, err = writeHelloWorld()
	assert.NoError(t, err)
	_, err = fetch("/api/v1/query?series=world&limit=1&offset=1&order=asc")
	assert.NoError(t, err)
}

func TestDelete(t *testing.T) {
	result, err := deleteHelloWorld()
	assert.NoError(t, err)
	assert.Contains(t, result, "\"status\":\"ok\"")
}

func TestQueryFail(t *testing.T) {
	_, err := fetch("/api/v1/query")
	assert.EqualError(t, err, "status code 400")
}

func TestDeleteFail(t *testing.T) {
	_, err := fetch("/api/v1/delete")
	assert.EqualError(t, err, "status code 400")
}

func TestDeleteByTime(t *testing.T) {
	_, err := fetch("/api/v1/write?series=world&time=111111111&value=hello")
	assert.NoError(t, err)
	result, err := fetch("/api/v1/deletebytime?series=world&time=111111111")
	assert.NoError(t, err)
	assert.Contains(t, result, "\"status\":\"ok\"")
}

func TestDeleteByTimeFail(t *testing.T) {
	_, err := fetch("/api/v1/deletebytime")
	assert.EqualError(t, err, "status code 400")
	_, err = fetch("/api/v1/deletebytime?series=world")
	assert.EqualError(t, err, "status code 400")
}

func TestCount(t *testing.T) {
	_, err := writeHelloWorld()
	assert.NoError(t, err)
	result, err := fetch("/api/v1/count?series=world")
	assert.NoError(t, err)
	assert.Contains(t, result, "\"result\":1")
	_, err = deleteHelloWorld()
	assert.NoError(t, err)
}

func TestCountFail(t *testing.T) {
	_, err := fetch("/api/v1/count")
	assert.EqualError(t, err, "status code 400")
}

func TestBackup(t *testing.T) {
	_, err := fetch("/backup")
	assert.NoError(t, err)
}

func TestShrink(t *testing.T) {
	_, err := fetch("/shrink")
	assert.NoError(t, err)
}

func TestMain(t *testing.T) {
	*flagLogLvl = "crit"
	go main()
}

func Benchmark_Write(b *testing.B) {
	for n := 0; n < b.N; n++ {
		if _, err := fetch("/api/v1/write?series=world&value=hello"); err != nil {
			b.Fail()
		}
	}
}

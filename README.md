# Lightweight Time Series Database

TSDB is lightweight in-memory time series database with [BuntDB](https://github.com/tidwall/buntdb) as backend

[![Build Status](https://travis-ci.org/semihalev/tsdb.svg)](https://travis-ci.org/semihalev/tsdb)
[![codecov](https://codecov.io/gh/semihalev/tsdb/branch/master/graph/badge.svg)](https://codecov.io/gh/semihalev/tsdb)
[![Go Report Card](https://goreportcard.com/badge/github.com/semihalev/tsdb)](https://goreportcard.com/report/github.com/semihalev/tsdb)
[![GoDoc](https://godoc.org/github.com/semihalev/tsdb?status.svg)](https://godoc.org/github.com/semihalev/tsdb)

## Warning
```
BoltDB backend changed. If you update latest version, migrate your data first.
```

## Features
+ HTTP API support

## Roadmap
- [x] Backend change to BuntDB
- [ ] Redis server support
- [ ] Raft support

## Usage

### Start using it

Download and install it:

```sh
$ go get github.com/semihalev/tsdb
```

```sh
$ go build
```

## API Usage

Query Series:
```
$ curl http://127.0.0.1:4080/api/v1/query?series=world (Optional parameters order=asc|desc, limit, offset)
```

Write Series:
```
$ curl http://127.0.0.1:4080/api/v1/write?series=world&value=hello (Optional parameters ttl=duration)
```

Count Series:
```
$ curl http://127.0.0.1:4080/api/v1/count?series=world
```

Delete Series:
```
$ curl http://127.0.0.1:4080/api/v1/delete?series=world
```

Delete by Time:
```
$ curl http://127.0.0.1:4080/api/v1/deletebytime?series=world&time=1435184955779847472
```

Backup:
```
$ curl http://127.0.0.1:4080/backup -o backup.db
```

## PHP Example Usage

- tsdb::query()
- tsdb::write()
- tsdb::count()
- tsdb::delete()
- tsdb::deletebytime()


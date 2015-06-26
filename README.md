# Small Time Series Database

TSDB is small NoSQL database with [BoltDB](https://github.com/boltdb/bolt) as backend

## Features
+ HTTP API support

## Build and Install
go get github.com/semihalev/tsdb

cd go/src/github.com/semihalev/tsdb

go build

## API Example

Query Series:
curl http://127.0.0.1:4080/api/v1/query?series=world

Write Series:
curl http://127.0.0.1:4080/api/v1/write?series=world&value=hello

Count Series
curl http://127.0.0.1:4080/api/v1/count?series=world

Delete Series:
curl http://127.0.0.1:4080/api/v1/delete?series=world

Delete by Time:
curl http://127.0.0.1:4080/api/v1/deletebytime?series=world&time=1435184955779847472

Backup DB:
curl http://127.0.0.1:4080/api/v1/backup

Stats DB:
curl http://127.0.0.1:4080/api/v1/stats

## PHP Client

tsdb::query()
tsdb::write()
tsdb::count()
tsdb::delete()
tsdb::deletebytime()


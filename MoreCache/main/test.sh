#!/bin/bash

trap "rm server; kill 0" EXIT           # 该shell进程收到EXIT信号后，删除server文件，kill 0代表杀死所有该进程组下的进程

go build -o server
./server -port=8001 &
./server -port=8002 &
./server -port=8003 -api=1 &

sleep 2
echo ">>> start test"
curl "http://localhost:9999/api?key=Tom" &
curl "http://localhost:9999/api?key=Tom" &
curl "http://localhost:9999/api?key=Tom" &

wait
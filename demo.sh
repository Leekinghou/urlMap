#!/bin/sh

gomake
 ./src -http=:8081 -rpc=true &
master_pid=$!
sleep 1
 ./src -master=127.0.0.1:8081 &
slave_pid=$!
echo "Running master on :8081, slave on :8080."
echo "Visit: http://localhost:8080/add"
echo "Press enter to shut down"
read
kill $master_pid
kill $slave_pid
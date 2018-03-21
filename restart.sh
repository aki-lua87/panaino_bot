#!/bin/sh

echo build
go build

echo kill
ps -ef | grep -E "panaino_bot" | grep -v grep | awk '{print "kill", $2}' | sh

echo start
nohup ./panaino_bot & 

echo finish

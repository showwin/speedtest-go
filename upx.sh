#!/bin/bash
set -ex

for binary in $(ls dist/speedtest-go*/speedtest-go*)
do
    upx --brute $binary &
done

wait
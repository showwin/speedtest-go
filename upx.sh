#!/bin/bash
set -ex

for binary in $(ls dist/speedtest-*/speedtest-*)
do
    upx --brute $binary &
done

wait

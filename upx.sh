#!/bin/bash
set -ex

for binary in $(dist/speedtest-go*/speedtest-go*)
do
    upx $binary &
done

wait
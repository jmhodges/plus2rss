#!/bin/bash

set -o errexit
set -o nounset

rm -rf release
go build .
mkdir -p release/usr/local/plus2rss
cp *template* release/usr/local/plus2rss
cp google_simple_key release/usr/local/plus2rss
cp plus2rss release/usr/local/plus2rss
mkdir -p release/etc/init
cp upstart/plus2rss.conf release/etc/init
fpm -s dir -t deb -n plus2rss -v 1.0 -C release .

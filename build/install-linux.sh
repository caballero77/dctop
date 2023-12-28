#!/bin/bash
cd "$(dirname "$0")"

mkdir -p /usr/local/bin 
cp -p bin/dctop /usr/local/bin/dctop

mkdir -p /usr/local/share/dctop
cp -pr themes /usr/local/share/dctop
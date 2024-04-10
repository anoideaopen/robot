#!/bin/sh

# This script generates autodoc for robot and puts it into godoc directory
# Requires godoc and wget

PID=$!
BASEDIR=$(dirname "$0")
GODOC="$BASEDIR"/godoc
GODOC_SRV="localhost:6060"

# Clear
rm -rf "$GODOC"/*
# Run godoc
godoc -http="$GODOC_SRV" & export PID=$!
# Wait for godoc started
sleep 2
# Getting all docs
wget -r -np -N -E -p -k http://"$GODOC_SRV"/pkg/github.com/anoideaopen/robot/ --directory-prefix="$GODOC"
# Move docs
mv  -v "$GODOC"/"$GODOC_SRV"/* "$GODOC"/
rmdir "$GODOC"/"$GODOC_SRV"
# Stop godoc
kill -9 $PID
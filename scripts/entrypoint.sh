#!/bin/sh
set -e
DATA_DIR=${DATA_DIR:-/app/data}
mkdir -p "$DATA_DIR"
exec /app/lanmapper

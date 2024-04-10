#!/usr/bin/env bash
if [[ "$1" == "" ]]; then
    echo "specify your own .env file"
    exit 1
fi

set -e
source ./"$1"

docker-compose -p robot down --remove-orphans --volumes --rmi all

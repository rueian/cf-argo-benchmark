#!/usr/bin/env bash

if [ -f "$GF_PATHS_DATA/grafana.db" ]; then
    echo "db exist"
else
    echo "init db"
    cp /grafana.db "$GF_PATHS_DATA"
fi

/run.sh
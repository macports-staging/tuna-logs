#!/bin/bash
# Usage: $0 201908{01..31}
set -eo pipefail

for log_date in "$@"; do
    for server in neo nano; do
        tuna_url="https://mirrors.tuna.tsinghua.edu.cn/logs/${server}mirrors/mirrors.log-$log_date.gz"
        echo "Downloading $tuna_url..."
        wget -O "$server.mirrors.log-$log_date.gz" "$tuna_url"
    done

    date=$(date -d "$log_date -1 day" +%Y%m%d)
    year="${date:: -4}"
    month="${date: -4:2}"
    dir="data/$year/$month"
    dest="mirrors.macports.log-$date.json.zst"

    mkdir -p "$dir"
    gzip -dc {neo,nano}".mirrors.log-$log_date.gz" | pv | go run ./cmd/tuna2json | zstd - -o "$dir/$dest"
    sha256sum --tag {neo,nano}".mirrors.log-$log_date.gz" >"$dir/checksum-$date.raw.sha256"
    rm {neo,nano}".mirrors.log-$log_date.gz"
done

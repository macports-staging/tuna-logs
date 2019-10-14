#!/bin/bash
set -eo pipefail

git clone -b gh-pages "https://github.com/$GITHUB_REPOSITORY.git" tuna-logs

date=$(date -d "2 days ago" +%Y%m%d)
year="${date:: -4}"
month="${date: -4:2}"
log_date=$(date -d yesterday +%Y%m%d)

dest="mirrors.macports.log-$date.json.zst"
if [ -f "tuna-logs/data/$year/$month/$dest" ]; then
    exit 0
fi

for server in neo nano; do
    tuna_url="https://mirrors.tuna.tsinghua.edu.cn/logs/${server}mirrors/mirrors.log-$log_date.gz"
    echo "Downloading $tuna_url..."
    wget --progress=dot:giga -O "$server.mirrors.log-$log_date.gz" "$tuna_url"
done

gzip -dc {neo,nano}".mirrors.log-$log_date.gz" | pv | go run ./cmd/tuna2json | zstd - -o "$dest"
sha256sum --tag {neo,nano}".mirrors.log-$log_date.gz" >"checksum-$date.raw.sha256"
rm {neo,nano}".mirrors.log-$log_date.gz"

if [ -s "$dest" ]; then
    mkdir -p "tuna-logs/data/$year/$month"
    cp "$dest" "tuna-logs/data/$year/$month/$dest"
    cp "checksum-$date.raw.sha256" "tuna-logs/data/$year/$month/"
    pushd tuna-logs
    git config user.email "41898282+github-actions[bot]@users.noreply.github.com"
    git config user.name "github-actions[bot]"
    git add "data/$year/$month"
    git commit -m "Auto commit"
    git push "https://x-access-token:$GITHUB_TOKEN@github.com/$GITHUB_REPOSITORY.git" gh-pages
    popd
fi

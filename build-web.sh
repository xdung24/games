#!/bin/sh

set -e

version=$(sed -n 's/^Version = "\([^"]*\)"$/\1/p' Games.toml)
if [ -z "$version" ]; then
	echo "Could not read Version from Games.toml" >&2
	exit 1
fi

GOOS=js GOARCH=wasm go build -tags 'release no_emoji migrated_fynedo' -ldflags='-s -w' -o web/games.wasm .

escaped_version=$(printf '%s' "$version" | sed 's/[&#\\]/\\&/g')
sed -i \
	-e "s#<div class=\"application-version\">v[^<]*</div>#<div class=\"application-version\">v$escaped_version</div>#" \
	-e "s#games[.]wasm?v=[A-Za-z0-9.+-]*#games.wasm?v=$escaped_version#" \
	web/index.html

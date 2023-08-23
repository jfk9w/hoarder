#!/usr/bin/env bash

[[ -z "$NAME" ]] && NAME=$(head -1 go.mod | cut -d ' ' -f2 | cut -d '/' -f3)
[[ -z "$VERSION" ]] && VERSION=$(git symbolic-ref -q --short HEAD || git describe --tags --exact-match || echo unknown)

DIR=dist

rm -rf $DIR/*

for GOOS in windows linux darwin; do
  for GOARCH in amd64 arm64; do
    make clean bin config
    if [ "$GOOS" == "windows" ]; then
      for FILE in bin/*; do
        mv "$FILE" "$FILE.exe"
      done
    fi

    DIST="$DIR/$NAME-$VERSION.$GOOS.$GOARCH"
    mkdir -p "$DIST"
    cp README.md LICENSE bin/* config/* "$DIST/"
    tar -zvcf "$DIST.tar.gz" -C "$(dirname "$DIST")" "$(basename "$DIST")"
    rm -rf "$DIST"
  done
done

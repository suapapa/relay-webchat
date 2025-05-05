#!/bin/bash

export ASSET_DIR=~/ws/homin-dev/homin-dev_asset

set -e

npm run build

cp dist/webchat-widget.css $ASSET_DIR/asset/css/
cp dist/webchat-widget.umd.js $ASSET_DIR/asset/script/

pushd $ASSET_DIR
git add asset/.
git commit -m "update webchat widget"
git push
popd


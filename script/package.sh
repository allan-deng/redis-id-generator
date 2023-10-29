#!/bin/bash
rm -rf ./idgen/
mkdir idgen

# bin
commitid=$(git rev-parse --short HEAD)
md5=$(md5sum idgensvr | awk '{print $1}')
filename='idgensvr_'$commitid'_'$md5
cp idgensvr ./idgen/$filename
cd idgen
ln -sf $filename idgen
cd ..
# conf
cp -r config ./idgen
# log
mkdir idgen/log
# archive
tar -zcvf idgensvr.tar.gz ./idgen

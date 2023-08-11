#!/bin/bash

testDir="test-dir-$(date +%s)"
mkdir -p "$testDir"

prepare_zip() {
  src=$1
  dst=$1.zip
  zip --symlinks -r $testDir/$dst $src
}

for dir in *
do
  [ -d $dir ] && prepare_zip $dir
done

exit 0

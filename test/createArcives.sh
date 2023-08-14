#!/bin/bash

testDir="test-dir-$(date +%s)"
mkdir -p "$testDir"

prepare_zip() {
  src=$1
  dst=$1.zip
  zip --symlinks -r $testDir/$dst $src

}

prepare_overwrite() {
  echo "file from upper dir"  > ../traversal
  zip -r 3_PathtraversalExtract.zip ../traversal
  rm ../traversal
}

for dir in *
do
  if [ "$dir" != "$testDir" ]
  then
    [ -d $dir ] && prepare_zip $dir
  fi
done

# create with ../ filepath
cd "$testDir"
prepare_overwrite
cd ..

# prepare 42.zip testcase
cp 42.zip $testDir/42.zip


exit 0

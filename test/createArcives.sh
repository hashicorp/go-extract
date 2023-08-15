#!/bin/bash

testDir="test-dir-$(date +%s)"
mkdir -p "$testDir"

prepare_zip() {
  mkdir -p "$testDir/zip"
  src=$1
  dst=$1.zip
  zip --symlinks -r $testDir/zip/$dst $src

}

prepare_tar() {
  mkdir -p "$testDir/tar"
  src=$1
  dst=$1.tar
  tar -cvf $testDir/tar/$dst $src
}

prepare_overwrite_zip() {
  echo "file from upper dir"  > ../traversal
  zip -r 3_PathtraversalExtract.zip ../traversal
  rm ../traversal

  echo "file from tmp" > /tmp/traversalextract
  zip -r 4_AbsolutExtract.zip /tmp/traversalextract
  rm /tmp/traversalextract
}

prepare_overwrite_tar() {
  echo "file from upper dir"  > ../traversal
  tar -rf 3_PathtraversalExtract.tar ../traversal
  rm ../traversal

  echo "file from tmp" > /tmp/traversalextract
  tar -rf 4_AbsolutExtract.tar /tmp/traversalextract
  rm /tmp/traversalextract
}

echo [i] create zip testcases

for dir in *
do
  if [ "$dir" != "$testDir" ]
  then
    [ -d $dir ] && prepare_zip $dir
  fi
done

cd "$testDir/zip"
prepare_overwrite_zip
cd ../../

echo [i] create tar testcases
for dir in *
do
  if [ "$dir" != "$testDir" ]
  then
    [ -d $dir ] && prepare_tar $dir
  fi
done

cd "$testDir/tar"
prepare_overwrite_tar
cd ../../


exit 0

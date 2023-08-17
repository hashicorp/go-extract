#!/bin/bash

cd test-dir-*

for type in *
do
  cd $type
  echo "-------------------------------"
  echo [i] $(pwd)
  echo "-------------------------------"
  for archive in *
  do
      echo "[!] ../../../extract -v $archive "
      ../../../extract -v $archive
  done
  
  cd ..

done


echo "################################"
echo "# Tests performed"
echo "################################"


ls -lR
ls -l /tmp/traversalextract
cd ..
rm -rf test-dir-*
rm /tmp/traversalextract

exit 0

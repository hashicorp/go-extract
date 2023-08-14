#!/bin/bash

cd test-dir-*

for zip in *.zip
do
    echo "[!] ../../extract -v $zip "
    ../../extract -v $zip 
done

echo "################################"
echo "# Tests performed"
echo "################################"


ls -lR
cd ..
rm -rf test-dir-*

exit 0
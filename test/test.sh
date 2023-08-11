#!/bin/bash

cd test-dir-*

for zip in *.zip
do
    ../../extract -v $zip -o $(basename $zip )
done

ls -lR
cd ..
rm -rf test-dir-*

exit 0
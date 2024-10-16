#!/bin/bash

TF_MODULE_BASE="https://github.com/terraform-aws-modules/terraform-aws-iam/archive/refs/tags/v5.34.0"
MODULE_FILE_NAME="terraform-aws-iam"
RANDOM_FILE="random-file"

# Create random file
dd if=/dev/random of=$RANDOM_FILE bs=512m count=1 iflag=fullblock

# Download and prepare module as tar.gz
wget $TF_MODULE_BASE.tar.gz -O $MODULE_FILE_NAME.tar.gz     # download
cp $MODULE_FILE_NAME.tar.gz $MODULE_FILE_NAME.big.tar.gz    # copy
gunzip $MODULE_FILE_NAME.big.tar.gz                         # gunzip
tar rf $MODULE_FILE_NAME.big.tar $RANDOM_FILE               # add random content
gzip $MODULE_FILE_NAME.big.tar                              # repack

# Download and prepare module as zip
wget $TF_MODULE_BASE.zip -O $MODULE_FILE_NAME.zip     # download
cp $MODULE_FILE_NAME.zip $MODULE_FILE_NAME.big.zip    # copy
zip -u $MODULE_FILE_NAME.big.zip $RANDOM_FILE         # add random content

# Remove random file
rm $RANDOM_FILE

exit 0

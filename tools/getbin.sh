#!/usr/bin/env bash
###############################################################################
#
# Get binary file from AWS S3
#
###############################################################################
echo Start to Download binary file of bitflyerspider from S3
aws s3 cp s3://artifacts-0/bitflyerspider/bin/linux-amd64/bitflyerspider ./
chmod 755 bitflyerspider
echo Finished

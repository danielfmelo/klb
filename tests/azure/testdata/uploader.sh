#!/usr/bin/env nash

import klb/azure/login
import klb/azure/blob/uploader

resgroup      = $ARGS[1]
location      = $ARGS[2]
accountname   = $ARGS[3]
sku           = $ARGS[4]
tier          = $ARGS[5]
containername = $ARGS[6]
remotepath    = $ARGS[7]
localpath     = $ARGS[8]

azure_login()

echo "uploading file"
uploader, err <= azure_blob_uploader_new(
	$resgroup,
	$location,
	$accountname,
	$sku,
	$tier,
	$containername
)
if $err != "" {
	echo "error creating uploader: " + $err
	exit("1")
}

err <= azure_blob_uploader_upload($uploader, $remotepath, $localpath)
if $err != "" {
	echo "error creating uploader: " + $err
	exit("1")
}
echo "success"

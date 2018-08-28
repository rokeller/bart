# bart - backup and restore tool

## Overview

`bart` is a simple backup/restore tool for data stored in a local file system. currently bart by default only supports backing up to Azure Blob Storage, though by compiling it with the `files` tag you can make it back up to the file system too.

`bart` supports the following:
* backup of local files
* restore of files missing locally but available in the archive
* password based AES encryption for the archive index and archived files

**Disclaimer: Use at your own risk!**

### Build

to build `bart` you can take advantage of the makefile in the repo:

```
make bart
# or, to also create the x86 and x64 binaries for Windows
make all
```

alternatively, you can just use `go build` yourself. to build `bart` with support to backup to the file system, run

```
TAGS=files make clean bart
```

## Usage

### Backup to Azure Blob Storage

```
./bart [-name string] [-path string] [-m noop|restore|delete] -acct string -key string
  -name string
        The name of the backup archive. (default "backup")
  -path string
        The path to the directory to backup and/or restore. (default ".")
  -m string
        A behavior for files missing locally: 'noop' to do nothing, 'restore' to restore them from the backup, 'delete' to delete them in the backup archive. (default "noop")
  -acct string
        The Azure Storage Account name.
  -key string
        The Azure Storage Account Key.
```
Example:
```
./bart -name my-pictures -path ~/Pictures -account example -key xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx/yyyyyyyyyyyyyyyyyyyy/zzzzzzzzzzzzzzzzzzzzzz==
```

### Backup to File System (built with build tag `files`)

```
./bart [-name string] [-path string] [-m noop|restore|delete] [-t string]
  -name string
        The name of the backup archive. (default "backup")
  -path string
        The path to the directory to backup and/or restore. (default ".")```
  -m string
        A behavior for files missing locally: 'noop' to do nothing, 'restore' to restore them from the backup, 'delete' to delete them in the backup archive. (default "noop")
  -t string
        The target root path for the backup. (default "$HOME/.backup")
```
Example:
```
./bart -name my-pictures -path ~/Pictures -t /mnt/mounted-network-drive/backups -m restore
```

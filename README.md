# bart - backup and restore tool

## Overview

`bart` is a simple backup/restore tool for data stored in a local file system.
Currently bart by default only supports backing up to Azure Blob Storage, though
by compiling it with the `files` tag you can make it back up to the file system
too.

`bart` supports the following:

* backup of local files
* restore of files missing locally but available in the archive
* password based AES encryption for the archive index and archived files
* concealing original file paths by hashing them

> **Disclaimer**: Use at your own risk!

## Usage

> **Note**: When downloading packages from a release, the binaries contained
> within are named `bart-<backupTarget>-<os>-<arch>` (and optionally with a file
> extension `.exe` for Windows). You can safely rename these executables to
> `bart` (or `bart.exe` on Windows) and call them with the usages outlined
> below, or you can call them just like they're unpacked.

### Backup to Azure Blob Storage

```text
Usage of ./bart:
  -azep string
        The blob service endpoint URL.
  -m string
        A behavior for files missing locally: 'noop' to do nothing, 'restore' to restore them from the backup, 'delete' to delete them in the backup archive. (default "noop")
  -name string
        The name of the backup archive. (default "backup")
  -p int
        The degree of parallelism to use. (default <depending on available CPUs>)
  -path string
        The path to the directory to backup and/or restore. (default ".")
```

Example:

```bash
./bart -name my-pictures -path ~/Pictures -azep 'https://myblobstorage.blob.core.windows.net/'
```

#### Authentication

`bart` currently uses default means for authentication with Azure to access the
blob storage services. The details of the different places where credentials are
looked for can be found in the [package documentation](https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/azidentity@v1.4.0).

To summarize, the following credentials are tried in order:

1. [Environment variables](https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/azidentity@v1.4.0#readme-environment-variables)
2. [Azure Workload identity](https://learn.microsoft.com/en-us/azure/active-directory/workload-identities/workload-identities-overview)
3. Managed identities
4. Azure CLI credential

When running `bart` in an Azure workload itself, #2 and #3 above may be easiest
to use. When running in a shell where you're already logged in to Azure CLI, #4
is probably best. Otherwise, using the environment variables, typically in
combination with a service principal may be easiest.

### Backup to File System (built with build tag `files`)

```text
Usage of ./bart:
  -m string
        A behavior for files missing locally: 'noop' to do nothing, 'restore' to restore them from the backup, 'delete' to delete them in the backup archive. (default "noop")
  -name string
        The name of the backup archive. (default "backup")
  -p int
        The degree of parallelism to use. (default <depending on available CPUs>)
  -path string
        The path to the directory to backup and/or restore. (default ".")
  -t string
        The target root path for the backup. (default "$HOME/.backup")
```

Example:

```bash
./bart -name my-pictures -path ~/Pictures -t /mnt/mounted-network-drive/backups -m restore
```

# bart - backup and restore tool

## Overview

`bart` is a simple backup/restore tool for data stored in a local file system.
Currently `bart` only supports backing up to Azure Storage blobs or to a file
system.

`bart` supports the following:

* backup of local files
* restore of files missing locally but available in the archive
* password based AES encryption for the archive index and archived files
* concealing original file paths in backup archives by hashing them

> **Disclaimer**: Use at your own risk!

## Usage

> **Note**: When downloading packages from a release, the binaries contained
> within are named `bart-<backupTarget>-<os>-<arch>` (and optionally with a file
> extension `.exe` for Windows) and they are packaged up in `.tar.gz` (Linux)
> or `.zip` files (Windows). You can safely rename these executables after
> extraction to `bart` (or `bart.exe` on Windows) and call them with the usages
> documented below, or you can call them just using their names from the packages.

`bart` has a few sub-commands:

* `backup` to run in backup mode, in which `bart` looks for files that are
  present locally, but not in the backup archive.
* `restore` to run in restore mode, where `bart` goes through all files found in
  the backup archive and checks if they're present locally too.
* `cleanup` to remove files in the backup archive or locally depending on the
  `-l` (location) flag.

You can get more information on the flags available for each sub-command by
running

```bash
bart <sub-command> --help
```

After parsing command line arguments and making sure everything is in order,
`bart` will ask you for a password. This password is used together with a _salt_
(randomly generated the first time a backup archive in a target store is touched)
to derive a symmetric encryption key. This key is used to encrypt each file
before it is uploaded in `backup` mode to the target store, or to decrypt each
file before it is downloaded in `restore` mode from the target store.

The password can either be entered by the user after the program has started,
or it can be piped into `bart` as follows. Any other means to pipe the password
works too, of course.

```bash
$ cat .password
my-password
$ cat .password | bart <sub-command>
```

### Target Azure Storage blobs

To use a backup archive stored in Azure Storage blobs, you must provide `bart`
with two pieces of information:

1. The blob service endpoint URL for an Azure Storage account, through the `-azep`
   flag on the command line. The value typically looks like 
   `https://your-storage-account-name.blob.core.windows.net/` where
   `your-storage-account-name` is the name of the storage account you want to
   use in Azure. The blob service endpoint URL can be found in the Azure portal
   under _Endpoints_ for the storage account in question. The DNS suffix may
   look different depending on the cloud (Public vs Government vs China etc.)
   you're using.
2. A credential to access the Azure Storage account. There are not command line
   switches to provide the credential, instead they must be passed in a way
   supported by the [Azure Identity Client module](https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/azidentity).
   The module will try to look for a credential using the following order:

   1. [Environment variables](https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/azidentity#readme-environment-variables)
   2. [Azure Workload identity](https://learn.microsoft.com/en-us/entra/workload-id/workload-identities-overview)
   3. Managed identities
   4. Azure CLI credential

   When running `bart` in an Azure workload itself, #2 and #3 above are best/easiest
   to use. When running in a shell where you're already logged in to Azure CLI, #4
   is probably best. Otherwise, using the environment variables, typically in
   combination with a service principal may be easiest.

### Target the file system

To use a backup archive stored in the file system itself, you must provide `bart`
with one additional piece of information:

1. The root directory where the backup files should be added, through the `-t`
   (target) flag on the command line.

## Build

To build `bart` by yourself you can take advantage of the `makefile` in the repo.

```bash
# build for your default OS and architecture for Azurite (Azure Storage emulator) target
make bart

# or, to build for Azure backup target
make bart TAGS=azure

# or, to also create the x86 and x64 binaries for Windows
make all
```

Alternatively, you can just use `go build -tags <TAGS>` yourself. The go build
tags used are described in the table below.

| Build tag | Meaning |
| --- | --- |
| `azure` | Target Azure Storage blobs for backup archives. |
| `azurite` | Target _Azurite_, the Azure Storage blobs emulator for backup archives. |
| `files` | Target the file system for backup archives. |

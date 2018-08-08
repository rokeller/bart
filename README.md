# bart - backup and restore tool

bart is a simple backup/restore tool for data stored in a local file system. currently bart only supports backing up to Azure Blob Storage, though there is code to also backup to the file system too -- it's just not hooked up.

bart supports the following:
* backup of local files
* restore of files missing locally but available in the archive
* password based AES encryption for the archive index and archived files

**Disclaimer: Use at your own risk!**

## Usage

```
./bart -name <archive-name> -path <local-path> [-m] -acct <azure-storage-account-name> -key <azure-storage-account-key>
  -name <archive-name>
        The name of the archive. Must follow the Azure Blob Container naming rules/restrictions.
  -path <local-path>
        The local path to a directory which should be backed up (or restored)
  -m    If set, restores files missing locally.
  -acct <azure-storage-acount-name>
        The name of the Azure Storage Account to which to backup the files.
  -key  <azure-storage-account-key>
        The key of the Azure Storage Account to which to backup the files.
```
Example:
```
./bart -name my-pictures -path ~/Pictures -account example -key xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx/yyyyyyyyyyyyyyyyyyyy/zzzzzzzzzzzzzzzzzzzzzz==
```

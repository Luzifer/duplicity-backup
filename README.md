[![License: Apache v2.0](https://badge.luzifer.io/v1/badge?color=5d79b5&title=license&text=Apache+v2.0)](http://www.apache.org/licenses/LICENSE-2.0)
[![Go Report Card](https://goreportcard.com/badge/github.com/Luzifer/duplicity-backup)](https://goreportcard.com/report/github.com/Luzifer/duplicity-backup)
[![Build Status](https://travis-ci.org/Luzifer/duplicity-backup.svg?branch=master)](https://travis-ci.org/Luzifer/duplicity-backup)

# Luzifer / duplicity-backup

`duplicity-backup` is a wrapper to execute a duplicity backup using a configuration file. It is designed to simplify handling backups on and restores from remote targets. All information required for the backup is set using the configuration file. Also the wrapper notifies targets (slack / [mondash](https://mondash.org/)) about successful and failed backups.

## Using without writing passwords to disk

Starting with version `v0.7.0` the `duplicity-backup` wrapper supports reading variables from the environment instead of writing the secrets to your disk. In every section of the file you can use the function `{{env "encrypt-password"}}` to read configuration options from the environment. As an example you could utilize [`vault2env`](https://gobuilder.me/github.com/Luzifer/vault2env) to set those variables from a Vault instance:

```bash
# vault write /secret/backups/mybackup encrypt-password=bVFq5jdyvkHD6VCvSQUY
Success! Data written to: secret/backups/mybackup

# cat ~/.duplicity.yaml
[...]
encryption:
  enable: true
  passphrase: {{env `encrypt-password`}}
[...]

# vault2env /secret/backups/mybackup -- duplicity-backup -f ~/.duplicity.yaml backup
(2016-06-25 15:07:06) ++++ duplicity-backup v0.7.0 started with command 'backup'
[...]
```

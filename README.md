[![Download on GoBuilder](http://badge.luzifer.io/v1/badge?title=Download%20on&text=GoBuilder)](https://gobuilder.me/github.com/Luzifer/duplicity-backup)
[![License: Apache v2.0](https://badge.luzifer.io/v1/badge?color=5d79b5&title=license&text=Apache+v2.0)](http://www.apache.org/licenses/LICENSE-2.0)
[![Go Report Card](https://goreportcard.com/badge/github.com/Luzifer/duplicity-backup)](https://goreportcard.com/report/github.com/Luzifer/duplicity-backup)

# Luzifer / duplicity-backup

`duplicity-backup` is a wrapper to execute a duplicity backup using a configuration file. It is designed to simplify handling backups on and restores from remote targets. All information required for the backup is set using the configuration file. Also the wrapper notifies targets (slack / [mondash](https://mondash.org/)) about successful and failed backups.

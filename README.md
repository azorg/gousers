Simple application to read records from utmp/wtmp/btmp linux files
==================================================================
Tested only on `amd64` (`x86_64`) platform.

## Build
```
$ make
```

## Install
```
$ sudo make install
```

## Uninstall
```
$ sudo make uninstall
```

## Help
```
$ gousers --help
gousers - simple dump for utmp/wtmp/btmp linux files
Usage: gousers [options] [command]

Options:
  -help|--help - print full help
  -h|--h       - print help about options only
  -f <file>    - use a specific file instead of /var/run/utmp
  -n           - notify mode (Ctrl+C to stop)

Commands:
  user[s]         - show users is currently logged (default command)
  dump            - show full dump
  info <username> - show full information about user by username (JSON)
  stat            - show logged user statistics (JSON)

Example:
  gousers --help                - print full help
  gousers [users]               - show users from /var/run/utmp
  gousers dump                  - dump /var/run/utmp
  gousers info alice            - show full information about user alice
  gousers stat                  - show logged user statistics
  gousers -f /var/log/btmp user - show users from /var/run/btmp
  gousers -f /var/log/wtmp dump - dump /var/log/wtmp
```


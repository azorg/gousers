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

## Unistall
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

Commands:
  user[s] - show users is currently logged (default command)
  stat    - show logged user statistics (JSON)
  dump    - show full dump

Example:
  gousers --help                - print full help
  gousers [users]               - show users from /var/run/utmp
  gousers stat                  - show logged user statistics
  gousers -f /var/log/btmp user - show users from /var/run/btmp
  gousers dump                  - dump /var/run/utmp
  gousers -f /var/log/wtmp dump - dump /var/log/wtmp
```

## Usage examples

### Show users from /var/run/utmp
```
$ gousers
2023-07-19 09:09:29 User='user' TTY='tty7' ID=':0' PID=1384 Host=':0' SID=0
```

### Show dump of /var/run/utmp
```
$ sudo gousers dump
2023-07-19 09:09:23 #2     REBOOT User='reboot' Kernel='6.1.0-10-amd64' SID=0
2023-07-19 09:09:29 #7  USER_PROC User='user' TTY='tty7' ID=':0' PID=1384 Host=':0' Term/Exit=0/0 SID=0
2023-07-19 09:09:32 #6 LOGIN_PROC User='LOGIN' TTY='tty1' ID='tty1' PID=1297 Term/Exit=0/0 SID=1297
2023-07-19 09:09:33 #1    RUN_LVL RL=5
2023-07-19 11:16:34 #7  USER_PROC User='user' TTY='pts/3' ID='ts/3' PID=23024 Term/Exit=0/0 SID=0
```

### Show dump of /var/log/btmp
```
$ sudo gousers -f /var/log/btmp dump
2023-07-18 12:52:45 #7  USER_PROC User='root' TTY='tty1' ID='tty1' PID=1288 Term/Exit=0/0 SID=1288
2023-07-18 20:49:03 #7  USER_PROC User='user' TTY='tty2' ID='tty2' PID=75431 Term/Exit=0/0 SID=75431
2023-07-18 23:54:35 #6 LOGIN_PROC User='root' TTY='pts/5' ID='5' PID=111464 Term/Exit=0/0 SID=0
```

### Show user statistics
```
$ gousers stat
{
  "total": 2,
  "local": 0,
  "remote": 1,
  "local_x": 1,
  "remote_x": 0,
  "local_root": true,
  "user": {
    "name": "user",
    "pid": 1384,
    "tty": "tty7",
    "host": ":0",
    "id": ":0",
    "time": "2023-07-19T09:09:29.41857+03:00"
  },
  "user_type": "local_x",
  "user_logons": 2
```


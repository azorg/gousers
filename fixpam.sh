#! /bin/bash
#
# Внести исправления в конфигурацию PAM для сохранения
# в /var/log/wtmp необходимых событий входа/выхода пользователей.

# Модифицируемые файлы
PAMFILES=`cat << EOF
/etc/pam.d/su
/etc/pam.d/su-l
/etc/pam.d/sudo
/etc/pam.d/sudo-i
/etc/pam.d/xrdp-sesman
EOF`

SPACE="\s*"
REGEXP="^${SAPCE}session${SPACE}.*${SPACE}pam_lastlog.so"
STRING=`cat << EOF

# Modify /var/log/wtmp by login/logout (DevLock)
session optional pam_lastlog.so silent
EOF`

echo "$PAMFILES" | while read F
do
  [ -f "$F" ] || continue
  if [ -z "$(grep "$REGEXP" "$F")" ]
  then
    echo "$STRING" | sudo tee -a "$F" > /dev/null && echo "fix '$F'"
  fi
done


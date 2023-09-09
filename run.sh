#! /bin/bash

PROC_NUM=`grep processor /proc/cpuinfo | wc -l`
[ "$PROC_NUM" -gt 4 ] && PROC_NUM="$(($PROC_NUM / 2))"
#echo "PROC_NUM=$PROC_NUM"

OLD_PWD=`pwd`
WORK_DIR=`dirname "$0"`
cd "$WORK_DIR"

make fmt && make -j "$PROC_NUM" all || exit

echo
echo "Users from utmp:"
bin/gousers user

echo
echo "Dump utmp:"
bin/gousers dump

echo
echo "Users from btmp:"
sudo bin/gousers -file /var/log/btmp

echo
echo "Dump btmp:"
sudo bin/gousers -file /var/log/btmp dump

#echo
#echo "Dump wtmp:"
#bin/gousers -file /var/log/wtmp dump

echo
echo "Users stat:"
bin/gousers stat

cd "$OLD_PWD"


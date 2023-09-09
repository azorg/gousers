#! /bin/bash

PROC_NUM=`grep processor /proc/cpuinfo | wc -l`
[ "$PROC_NUM" -gt 4 ] && PROC_NUM="$(($PROC_NUM / 2))"
#echo "PROC_NUM=$PROC_NUM"

OLD_PWD=`pwd`
WORK_DIR=`dirname "$0"`
cd "$WORK_DIR"

make fmt && make -j "$PROC_NUM" || exit 1

cd "$OLD_PWD"


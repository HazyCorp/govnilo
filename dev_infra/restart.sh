#!/bin/bash
function restart () {
  for KILLPID in `ps ax | grep $1 | awk ' { print $1;}'`; do
    echo $KILLPID
    kill -9 $KILLPID &> /dev/null || true
  done

  ./$1 &> /$1.log &
}

mkdir ./logs || true

declare -a services=(
    "minion"
)

for service in "${services[@]}"
do
  restart $service
done


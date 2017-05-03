#!/bin/bash

# show message function
function show() {
    echo -e "\n===================================================="
    echo -e " $1"
    echo -e "====================================================\n"
}

function killServer() {

    if [ "$1" != "" ]; then
        kill -9 $1
    fi

    pid="`ps ax | grep -v grep | grep chat/server | awk '{print $1}'`"
    if [ "$pid" != "" ]; then
        kill -9 $pid
    fi
}

# launch
show "Launch server 2..."
./serverBully 2 &
PID_SERVER_2=$!
sleep 4s

show "Launch server 1..."
./serverBully 1 &
PID_SERVER_1=$!
sleep 4s

show "Launch server 4..."
./serverBully 4 &
PID_SERVER_4=$!
sleep 4s

show "Launch server 3..."
./serverBully 3 &
PID_SERVER_3=$!
sleep 4s


show "Wait 8 seconds..."
sleep 8s
show "Kill Server 4..."
killServer $PID_SERVER_4
sleep 4s
show "Start Server 4..."
./serverBully 4 &
PID_SERVER_4=$!


# wait
echo
echo "============================="
echo "PRESS [ENTER] TO STOP CLUSTER"
echo "============================="
echo

read foo

# kill
show "Kill processes: $PID_SERVER_1 $PID_SERVER_2 $PID_SERVER_3 $PID_SERVER_4"
killServer $PID_SERVER_1
killServer $PID_SERVER_2
killServer $PID_SERVER_3
killServer $PID_SERVER_4

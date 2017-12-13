#!/bin/bash
# test.sh
 
MAX=1000
 
for (( i = 0; i < MAX ; i ++ ))
do
    kbang -c 10 -n 4000  http://127.0.0.1:5000/test
    sleep 1
done
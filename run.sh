#!/usr/bin/bash

NGINX_VANILLA="./nginx-vanilla"
NGINX_TEMPFILES="./nginx"

function runTest {
        test_name=$1
        nginx=$2
        conf_file=$3
        file_time=$4
        run_time=$5
        run_paralel=$6

        echo "----------------------------------------------------------"
        echo "Test $test_name"
        echo "----------------------------------------------------------"

        echo -n "$test_name;" >> origin_stats.csv
        echo -n "$test_name;" >> client_stats.csv

        rm -rf "/tmp/nginx-benchmark/"
        mkdir "/tmp/nginx-benchmark/"

        $nginx -c "$conf_file" -e /dev/null
        trap "$nginx -c \"$conf_file\" -e /dev/null -s stop" RETURN

        # Origin (will be automatically ended when client stops)
        go run origin/main.go -time $file_time &
        sleep 1

        # Run tests
        go run client/main.go -t $run_time -p $run_paralel
        echo
}

function testSuite {
        suite_name=$1
        file_time=$2
        run_time=$3
        run_paralel=$4

        echo "=========================================================="
        echo "Test suite: $suite_name"
        echo "=========================================================="

        echo "$suite_name;requests;bytes;avg Mbps" >> origin_stats.csv
        echo "$suite_name;requests;bytes;avg Mbps;Avg byte wait time;Avg time to first byte" >> client_stats.csv

        runTest "1 - Vanilla nginx" $NGINX_VANILLA $(pwd)/nginx_plain.conf $file_time $run_time $run_paralel
        sleep 1
        runTest "2 - Vanilla nginx with cache lock" $NGINX_VANILLA $(pwd)/nginx_lock.conf $file_time $run_time $run_paralel
        sleep 1
        runTest "3 - Patched nginx with cache lock" $NGINX_TEMPFILES $(pwd)/nginx_lock.conf $file_time $run_time $run_paralel
        sleep 1
        runTest "4 - Patched nginx with cache tempfiles" $NGINX_TEMPFILES $(pwd)/nginx_tempfiles.conf $file_time $run_time $run_paralel

        trap "" RETURN
}

testSuite "Full speed origin (file_time: 0)" 0 30 100
testSuite "Slow speed origin (file_time: 1)" 1 30 100
testSuite "Slow speed origin (file_time: 1)" 3 30 100

join -t";" origin_stats.csv client_stats.csv >> stats.csv

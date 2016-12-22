#!/bin/bash

# 检测是否需要启动日志收集服务
if [[ -n ${IS_MONITOR} ]];then
  /bin/tcm -i eth0,eth1 
fi

# 启动grproxy
if ${GD_ADAPTER};then
   nohup /tmp/grproxy --logtostderr=false --log_dir=/tmp --bind-address=127.0.0.1 --master=http://172.30.42.1:8080 --healthz_port=0 --namespace=${TENANT_ID} > /dev/null 2>&1 &
fi

touch /tmp/log.log

# 检查服务启动的端口是否正常
echo "monitor port is ${MONITOR_PORT}"
if [[ -n ${MONITOR_PORT} ]]; then
    MAXWAIT=${MAXWAIT:-60}
    wait=0
    while [ ${wait} -lt ${MAXWAIT} ]
    do
        echo stat | nc 127.0.0.1 ${MONITOR_PORT}
        if [ $? -eq 0 ];then
            curl --header "Content-Type:application/json" -H "Authorization:Token 5ca196801173be06c7e6ce41d5f7b3b8071e680a" -X POST -d '{"pod_order":"'$POD_ORDER'","status":"running"}' http://region.goodrain.me:8888/v1/services/lifecycle/$SERVICE_ID/monitor-port/
            break
        fi
        wait=`expr ${wait} + 1`;
        echo "Waiting service ${wait} seconds"
        sleep 5
    done
    if [ "${wait}" =  ${MAXWAIT} ]; then
        echo >&2 'start failed, please ensure service has started.'
        curl --header "Content-Type:application/json" -H "Authorization:Token 5ca196801173be06c7e6ce41d5f7b3b8071e680a" -X POST -d '{"pod_order":"'$POD_ORDER'","status":"unknown"}' http://region.goodrain.me:8888/v1/services/lifecycle/$SERVICE_ID/monitor-port/
    fi

fi

tail -f /tmp/log.log
#!/bin/bash

#123 验证组  4提案节点

#限制Arena内存池个数，控制虚拟内存消耗
export MALLOC_ARENA_MAX=4

instance_index=$1
instance_count=$2
instance_end=$instance_index+$instance_count

for((;instance_index<instance_end;instance_index++))

do
	if [ ! -d 'logs' ]; then
		mkdir logs
	fi

	if [ ! -d 'pid' ]; then
		mkdir pid
	fi

	rpc_port=$[8100+$instance_index]
	pprof_port=$[9000+$instance_index]
	config_file='x'$instance_index'.ini'

	stdout_log='logs/nohup_out_'$instance_index'.log'
	pid_file='pid/pid_gx'$instance_index'.txt'
    if [ -e $pid_file ];then
		kill -9 `cat $pid_file`
	fi

	if [ $instance_index -le 3 ];then
		nohup env GOTRACEBACK=crash ./rocket_node miner --config $config_file --rpc --rpcport $rpc_port  --instance $instance_index --pprof $pprof_port  --gateaddr 47.96.99.105:8888 > $stdout_log 2>&1 & echo $! > $pid_file
    elif [ $instance_index -eq 4 ];then
		nohup env GOTRACEBACK=crash ./rocket_node miner --config $config_file --rpc --rpcport $rpc_port  --instance $instance_index --pprof $pprof_port  --gateaddr 47.96.99.105:8888 > $stdout_log 2>&1 & echo $! > $pid_file
	else
		nohup env GOTRACEBACK=crash ./rocket_node miner --config $config_file --rpc --rpcport $rpc_port  --instance $instance_index --pprof $pprof_port  --gateaddr 47.96.99.105:8888 > $stdout_log 2>&1 & echo $! > $pid_file
	fi
	sleep 1
done

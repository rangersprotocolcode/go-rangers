#!/bin/bash

#123 验证组  456提案节点

#限制Arena内存池个数，控制虚拟内存消耗
export MALLOC_ARENA_MAX=4

instance_index=$1
instance_count=$2
instance_end=$instance_index+$instance_count

seed='192.168.3.69'
seedId='QmTn5a8UhdgmNZx1Vy82kNwJ6RmHKcqocSjTg8VbPaXu69'

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

	if [ $instance_index -eq 1 ];then
		nohup ./gx miner --config $config_file --rpc --rpcport $rpc_port --super --instance $instance_index --pprof $pprof_port  --seed $seed --seedId $seedId --apply light --keystore keystore$instance_index > $stdout_log 2>&1 & echo $! > $pid_file
    elif [ $instance_index -le 3 ];then
		nohup ./gx miner --config $config_file --rpc --rpcport $rpc_port  --instance $instance_index --pprof $pprof_port  --seed $seed --seedId $seedId --apply light --keystore keystore$instance_index > $stdout_log 2>&1 & echo $! > $pid_file
	else
		nohup ./gx miner --config $config_file --rpc --rpcport $rpc_port  --instance $instance_index --pprof $pprof_port --seed $seed --seedId $seedId --apply heavy --keystore keystore$instance_index > $stdout_log 2>&1 & echo $! > $pid_file
	fi
	sleep 1
done

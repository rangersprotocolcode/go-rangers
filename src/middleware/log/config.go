// Copyright 2020 The RocketProtocol Authors
// This file is part of the RocketProtocol library.
//
// The RocketProtocol library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The RocketProtocol library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the RocketProtocol library. If not, see <http://www.gnu.org/licenses/>.

package log

const (
	DefaultConfig = `<seelog minlevel="debug">
						<outputs formatid="default">
							<rollingfile type="size" filename="./logs/defaultLOG_INDEX.log" maxsize="100000000" maxrolls="1"/>
						</outputs>
						<formats>
							<format id="default" format="%Date(2006-01-02 15:04:05.000)[%File:%Line] %Msg%n" />
						</formats>
					</seelog>`

	CoreLogConfig = `<seelog minlevel="debug">
						<outputs formatid="default">
							<rollingfile type="size" filename="./logs/coreLOG_INDEX.log" maxsize="100000000" maxrolls="1"/>
						</outputs>
						<formats>
							<format id="default" format="%Date(2006-01-02 15:04:05.000)[%File:%Line]%Msg%n" />
						</formats>
					</seelog>`

	SyncLogConfig = `<seelog minlevel="debug">
						<outputs formatid="default">
							<rollingfile type="size" filename="./logs/syncLOG_INDEX.log" maxsize="100000000" maxrolls="1"/>
						</outputs>
						<formats>
							<format id="default" format="%Date(2006-01-02 15:04:05.000)[%LEV][%File:%Line]%Msg%n" />
						</formats>
					</seelog>`

	SyncHandleLogConfig = `<seelog minlevel="debug">
						<outputs formatid="default">
							<rollingfile type="size" filename="./logs/sync_handleLOG_INDEX.log" maxsize="100000000" maxrolls="1"/>
						</outputs>
						<formats>
							<format id="default" format="%Date(2006-01-02 15:04:05.000)[%LEV][%File:%Line]%Msg%n" />
						</formats>
					</seelog>`

	LockLogConfig = `<seelog minlevel="debug">
						<outputs formatid="default">
							<rollingfile type="size" filename="./logs/lockLOG_INDEX.log" maxsize="100000000" maxrolls="1"/>
						</outputs>
						<formats>
							<format id="default" format="%Date(2006-01-02 15:04:05.000)[%File:%Line]%Msg%n" />
						</formats>
					</seelog>`

	P2PLogConfig = `<seelog minlevel="debug">
						<outputs formatid="default">
							<rollingfile type="size" filename="./logs/p2pLOG_INDEX.log" maxsize="100000000" maxrolls="1"/>
						</outputs>
						<formats>
							<format id="default" format="%Date(2006-01-02 15:04:05.000)[%File:%Line]%Msg%n" />
						</formats>
					</seelog>`

	P2PBizLogConfig = `<seelog minlevel="debug">
						<outputs formatid="default">
							<rollingfile type="size" filename="./logs/p2p_bizLOG_INDEX.log" maxsize="100000000" maxrolls="1"/>
						</outputs>
						<formats>
							<format id="default" format="%Date(2006-01-02 15:04:05.000)[%File:%Line]%Msg%n" />
						</formats>
					</seelog>`

	ConsensusLogConfig = `<seelog minlevel="debug">
						<outputs formatid="default">
							<rollingfile type="size" filename="./logs/consensusLOG_INDEX.log" maxsize="50000000" maxrolls="1"/>
						</outputs>
						<formats>
							<format id="default" format="%Date(2006-01-02 15:04:05.000)[%File:%Line]%Msg%n" />
						</formats>
					</seelog>`

	GroupCreateLogConfig = `<seelog minlevel="debug">
						<outputs formatid="default">
							<rollingfile type="size" filename="./logs/group_createLOG_INDEX.log" maxsize="50000000" maxrolls="1"/>
						</outputs>
						<formats>
							<format id="default" format="%Date(2006-01-02 15:04:05.000)[%File:%Line]%Msg%n" />
						</formats>
					</seelog>`

	StdConsensusLogConfig = `<seelog minlevel="debug">
						<outputs formatid="default">
							<rollingfile type="size" filename="./logs/std_consensusLOG_INDEX.log" maxsize="50000000" maxrolls="1"/>
						</outputs>
						<formats>
							<format id="default" format="%Date(2006-01-02 15:04:05.000)[%File:%Line]%Msg%n" />
						</formats>
					</seelog>`
	GroupLogConfig = `<seelog minlevel="error">
						<outputs formatid="default">
							<rollingfile type="size" filename="./logs/groupLOG_INDEX.log" maxsize="50000000" maxrolls="1"/>
						</outputs>
						<formats>
							<format id="default" format="%Date(2006-01-02 15:04:05.000)[%File:%Line]%Msg%n" />
						</formats>
					</seelog>`
	StatisticsLogConfig = `<seelog minlevel="error">
						<outputs formatid="default">
							<rollingfile type="size" filename="./logs/statisticsLOG_INDEX.log" maxsize="50000000" maxrolls="1"/>
						</outputs>
						<formats>
							<format id="default" format="%Date(2006-01-02 15:04:05.000)[%File:%Line]%Msg%n" />
						</formats>
					</seelog>`
	VRFerrorLogConfig = `<seelog minlevel="error">
						<outputs formatid="default">
							<rollingfile type="size" filename="./logs/vrf_errorLOG_INDEX.log" maxsize="100000000" maxrolls="1"/>
						</outputs>
						<formats>
							<format id="default" format="%Date(2006-01-02 15:04:05.000)[%File:%Line]%Msg%n" />
						</formats>
					</seelog>`
	StateMachineLogConfig = `<seelog minlevel="debug">
						<outputs formatid="default">
							<rollingfile type="size" filename="./logs/state_machineLOG_INDEX.log" maxsize="50000000" maxrolls="1"/>
						</outputs>
						<formats>
							<format id="default" format="%Date(2006-01-02 15:04:05.000)[%File:%Line]%Msg%n" />
						</formats>
					</seelog>`
	MiddlewareLogConfig = `<seelog minlevel="error">
						<outputs formatid="default">
							<rollingfile type="size" filename="./logs/middlewareLOG_INDEX.log" maxsize="100000000" maxrolls="1"/>
						</outputs>
						<formats>
							<format id="default" format="%Date(2006-01-02 15:04:05.000)[%File:%Line]%Msg%n" />
						</formats>
					</seelog>`
	ForkLogConfig = `<seelog minlevel="debug">
						<outputs formatid="default">
							<rollingfile type="size" filename="./logs/forkLOG_INDEX.log" maxsize="50000000" maxrolls="1"/>
						</outputs>
						<formats>
							<format id="default" format="%Date(2006-01-02 15:04:05.000)[%File:%Line]%Msg%n" />
						</formats>
					</seelog>`
	SlowLogConfig = `<seelog minlevel="error">
						<outputs formatid="default">
							<rollingfile type="size" filename="./logs/slow_logLOG_INDEX.log" maxsize="50000000" maxrolls="1"/>
						</outputs>
						<formats>
							<format id="default" format="%Date(2006-01-02 15:04:05.000)|%Msg%n" />
						</formats>
					</seelog>`
	TxLogConfig = `<seelog minlevel="trace">
						<outputs formatid="default">
							<rollingfile type="size" filename="./logs/txLOG_INDEX.log" maxsize="100000000" maxrolls="1"/>
						</outputs>
						<formats>
							<format id="default" format="%Date(2006-01-02 15:04:05.000)[%File:%Line]%Msg%n" />
						</formats>
					</seelog>`
	AccountLogConfig = `<seelog minlevel="trace">
						<outputs formatid="default">
							<rollingfile type="size" filename="./logs/accountLOG_INDEX.log" maxsize="100000000" maxrolls="1"/>
						</outputs>
						<formats>
							<format id="default" format="%Date(2006-01-02 15:04:05.000)[%File:%Line]%Msg%n" />
						</formats>
					</seelog>`
	AccountDBLogConfig = `<seelog minlevel="debug">
						<outputs formatid="default">
							<rollingfile type="size" filename="./logs/accountDBLOG_INDEX.log" maxsize="100000000" maxrolls="1"/>
						</outputs>
						<formats>
							<format id="default" format="%Date(2006-01-02 15:04:05.000)[%File:%Line]%Msg%n" />
						</formats>
					</seelog>`
	TxPoolLogConfig = `<seelog minlevel="trace">
						<outputs formatid="default">
							<rollingfile type="size" filename="./logs/tx_poolLOG_INDEX.log" maxsize="100000000" maxrolls="1"/>
						</outputs>
						<formats>
							<format id="default" format="%Date(2006-01-02 15:04:05.000)[%File:%Line]%Msg%n" />
						</formats>
					</seelog>`
	PerformanceLogConfig = `<seelog minlevel="info">
						<outputs formatid="default">
							<rollingfile type="size" filename="./logs/performanceLOG_INDEX.log" maxsize="200000000" maxrolls="1"/>
						</outputs>
						<formats>
							<format id="default" format="%Date(2006-01-02 15:04:05.000)[%File:%Line]%Msg%n" />
						</formats>
					</seelog>`

	GameExecutorLogConfig = `<seelog minlevel="debug">
						<outputs formatid="default">
							<rollingfile type="size" filename="./logs/gameExecutorLOG_INDEX.log" maxsize="200000000" maxrolls="1"/>
						</outputs>
						<formats>
							<format id="default" format="%Date(2006-01-02 15:04:05.000)[%File:%Line]%Msg%n" />
						</formats>
					</seelog>`
	RPCLogConfig = `<seelog minlevel="debug">
						<outputs formatid="default">
							<rollingfile type="size" filename="./logs/rpcLOG_INDEX.log" maxsize="200000000" maxrolls="1"/>
						</outputs>
						<formats>
							<format id="default" format="%Date(2006-01-02 15:04:05.000)[%File:%Line]%Msg%n" />
						</formats>
					</seelog>`

	AccessLogConfig = `<seelog minlevel="debug">
						<outputs formatid="default">
							<rollingfile type="size" filename="./logs/consensus_accessLOG_INDEX.log" maxsize="50000000" maxrolls="1"/>
						</outputs>
						<formats>
							<format id="default" format="%Date(2006-01-02 15:04:05.000)[%File:%Line]%Msg%n" />
						</formats>
					</seelog>`
	RewardLogConfig = `<seelog minlevel="debug">
						<outputs formatid="default">
							<rollingfile type="size" filename="./logs/rewardLOG_INDEX.log" maxsize="100000000" maxrolls="1"/>
						</outputs>
						<formats>
							<format id="default" format="%Date(2006-01-02 15:04:05.000)[%File:%Line]%Msg%n" />
						</formats>
					</seelog>`
	RefundLogConfig = `<seelog minlevel="debug">
						<outputs formatid="default">
							<rollingfile type="size" filename="./logs/refundLOG_INDEX.log" maxsize="100000000" maxrolls="1"/>
						</outputs>
						<formats>
							<format id="default" format="%Date(2006-01-02 15:04:05.000)[%File:%Line]%Msg%n" />
						</formats>
					</seelog>`

	GroupCreateDebugLogConfig = `<seelog minlevel="debug">
						<outputs formatid="default">
							<rollingfile type="size" filename="./logs/group_create_debugLOG_INDEX.log" maxsize="50000000" maxrolls="1"/>
						</outputs>
						<formats>
							<format id="default" format="%Date(2006-01-02 15:04:05.000)[%File:%Line]%Msg%n" />
						</formats>
					</seelog>`
	LdbLogConfig = `<seelog minlevel="debug">
						<outputs formatid="default">
							<rollingfile type="size" filename="./logs/ldbLOG_INDEX.log" maxsize="50000000" maxrolls="1"/>
						</outputs>
						<formats>
							<format id="default" format="%Date(2006-01-02 15:04:05.000)[%File:%Line]%Msg%n" />
						</formats>
					</seelog>`

	VMLogConfig = `<seelog minlevel="debug">
						<outputs formatid="default">
							<rollingfile type="size" filename="./logs/vmLOG_INDEX.log" maxsize="100000000" maxrolls="1"/>
						</outputs>
						<formats>
							<format id="default" format="%Date(2006-01-02 15:04:05.000)[%File:%Line]%Msg%n" />
						</formats>
					</seelog>`
	ETHRPCLogConfig = `<seelog minlevel="debug">
						<outputs formatid="default">
							<rollingfile type="size" filename="./logs/eth_rpcLOG_INDEX.log" maxsize="100000000" maxrolls="1"/>
						</outputs>
						<formats>
							<format id="default" format="%Date(2006-01-02 15:04:05.000)[%File:%Line]%Msg%n" />
						</formats>
					</seelog>`
	MonitorLogConfig = `<seelog minlevel="debug">
						<outputs formatid="default">
							<rollingfile type="size" filename="./logs/monitorLOG_INDEX.log" maxsize="300000000" maxrolls="1"/>
						</outputs>
						<formats>
							<format id="default" format="%Date(2006-01-02 15:04:05.000)[%File:%Line]%Msg%n" />
						</formats>
					</seelog>`
)

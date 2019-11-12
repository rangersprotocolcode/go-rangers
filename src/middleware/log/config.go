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

	CoreLogConfig = `<seelog minlevel="info">
						<outputs formatid="default">
							<rollingfile type="size" filename="./logs/coreLOG_INDEX.log" maxsize="100000000" maxrolls="1"/>
						</outputs>
						<formats>
							<format id="default" format="%Date(2006-01-02 15:04:05.000)[%File:%Line]%Msg%n" />
						</formats>
					</seelog>`

	BlockSyncLogConfig = `<seelog minlevel="error">
						<outputs formatid="default">
							<rollingfile type="size" filename="./logs/block_syncLOG_INDEX.log" maxsize="100000000" maxrolls="1"/>
						</outputs>
						<formats>
							<format id="default" format="%Date(2006-01-02 15:04:05.000)[%File:%Line]%Msg%n" />
						</formats>
					</seelog>`

	GroupSyncLogConfig = `<seelog minlevel="error">
						<outputs formatid="default">
							<rollingfile type="size" filename="./logs/group_syncLOG_INDEX.log" maxsize="100000000" maxrolls="1"/>
						</outputs>
						<formats>
							<format id="default" format="%Date(2006-01-02 15:04:05.000)[%File:%Line]%Msg%n" />
						</formats>
					</seelog>`

	LockLogConfig = `<seelog minlevel="error">
						<outputs formatid="default">
							<rollingfile type="size" filename="./logs/lockLOG_INDEX.log" maxsize="100000000" maxrolls="1"/>
						</outputs>
						<formats>
							<format id="default" format="%Date(2006-01-02 15:04:05.000)[%File:%Line]%Msg%n" />
						</formats>
					</seelog>`

	P2PLogConfig = `<seelog minlevel="warn">
						<outputs formatid="default">
							<rollingfile type="size" filename="./logs/p2pLOG_INDEX.log" maxsize="100000000" maxrolls="1"/>
						</outputs>
						<formats>
							<format id="default" format="%Date(2006-01-02 15:04:05.000)[%File:%Line]%Msg%n" />
						</formats>
					</seelog>`

	ConsensusLogConfig = `<seelog minlevel="error">
						<outputs formatid="default">
							<rollingfile type="size" filename="./logs/consensusLOG_INDEX.log" maxsize="50000000" maxrolls="1"/>
						</outputs>
						<formats>
							<format id="default" format="%Date(2006-01-02 15:04:05.000)[%File:%Line]%Msg%n" />
						</formats>
					</seelog>`

	StdConsensusLogConfig = `<seelog minlevel="error">
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
	StateMachineLogConfig = `<seelog minlevel="error">
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
	TxLogConfig = `<seelog minlevel="error">
						<outputs formatid="default">
							<rollingfile type="size" filename="./logs/txLOG_INDEX.log" maxsize="100000000" maxrolls="1"/>
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

	DockerLogConfig = `<seelog minlevel="info">
						<outputs formatid="default">
							<rollingfile type="size" filename="./logs/dockerLOG_INDEX.log" maxsize="200000000" maxrolls="1"/>
						</outputs>
						<formats>
							<format id="default" format="%Date(2006-01-02 15:04:05.000)[%File:%Line]%Msg%n" />
						</formats>
					</seelog>`

	GameExecutorLogConfig = `<seelog minlevel="error">
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

)

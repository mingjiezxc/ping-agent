# ping-agent

## remark
   ping 是网络监控最基础监控方式。多IP可以使用 fping， 为什么写这个？
1. 新手练手
2. 精细化 ping 监控 10s(正常状态) / 1s（错误状态）
3. 性能优化（fping需等待结束,有IP数量限制，这个是以命令行的方式，c调用应该无这问题。）
4. 一般 ping 几千个IP,已开始有性能问题.
5. zabbix 作为后端。

-------------------------------------------------------------

   基础操作方式：
1. 程序根据 cron job 中的 ip list 发送 icmp 包, 包中包含发送时的 UnixNano
2. 对端返回 icmp pack ，将包中的 UnixNano 与 time.Now() 相减获得 ping值
3. 将 ping值 写入 redis ,并设置 timeout。
4. 如 redis key timeout，agent 将该Ip加入 err list 
5. cron job 定时检查 memory 中的 ip status，如无收到包 或 ping timeout，将IP 加入 err list
6. 每分钟检查 err list 掉包率,如低于检查值从 err list 移走 ip

## redis key

```bash
// 所有 agent 列表，agent 自写入
key: agent-list
type: set
value: [ "gza", "gzc" ]

// agent 在线状态
key: agent-online_$ageent
type: string & timeout
value: online

// ping is err list 表, 定时检查 ip 时间段内是否有丢包，如无则移出。
key: agent-err-list_$agent
type: set
value: ["192.168.1.2", "192.168.1.1",  "192.168.2.1"]

// ip 最新 ms 值 , agent 监听 timeout 事件，如发生将 ip ，添加至 err list
key: agent-ip-last-ms_$agent_$ip
type: string
value: 11

// ip 状态 
key: agent-all-ip-status_$agent
type: string
value: 

type IPStatus struct {
   // agent 名称
	Agent        string
   // ip address
	IP           string
   // redis key time out
	PTLL         int64
   // 在 err list 时 ErrIPRemoveJobSpec 允许掉多少个包
	AllowedLoss  int64
   // ErrIPRemoveJobSpec 发送 icmp 包数量
	SendCount    int64
   // ErrIPRemoveJobSpec 收到 icmp 包数量
	ReceiveCount int64
   // 是否在 err list
	InErrList    bool
   // ErrIPRemoveJobSpec ping的 ms 值
	Ms           []int64
   // ms 的平均值 
	MsAvg        int64
   // ErrIPRemoveJobSpec 丢包率
	Loss         string
   // ErrIPRemoveJobSpec 丢包数量
	Lost         int64
   // 收到最后一个 icmp 的时间
	UpdateTime   int64
}



// agent 的错误IP 列表
key: agent-err-list_$agent
type: set
value: ["192.16.111.1"]


// cron job list 名称
key: job-list
type: set 
value: ["xxx", "kkk", "ccc"]

// job status
key: job_$jobname
type: string
value: 
type JobStruct struct {
   // job 循环周期
	SPEC  string   `json:"spec" example:"*/10 * * * * *"`
   // job name
	Name  string   `json:"name" example:"job1"`
   // use ip group
	Group []string `json:"group" example:"group1,group2"`
   // redis key time out 
	PTTL  int64    `json:"pttl" example:"21"`
   // 计算周期(ErrIPRemoveJobSpec)内允许掉多少个包，认为是已恢复，并将ip 从 err list中移除
   AllowedLoss int64 `json:"allowedloss" example:"1"`
}

// ip group list
key: group-list
type: set
value: ["group1","ccc","bbb"]

// ip group 
key: group_$groupName
type: set
value: ["192.18.1.1", "192.168.2.1"]

```


## agent Listen ICMP package
[![](https://mermaid.ink/img/eyJjb2RlIjoiZ3JhcGggVERcbiAgICBBW0xpc3RlbiBJQ01QIFNlcnZlcl0gLS0-fGdldCBpY21wIHBhY2thZ2V8IEIocmVhZCBwYWNrYWdlIGpvYilcbiAgICBCIC0tPnxtZW1vcnkgbWFwfCBGW2dldCBpcCB0dGxdXG4gICAgQiAtLT58aXAgaGVhZHwgRFtzcmMgaXBdXG4gICAgQiAtLT58cGFja2FnZSBib2R5fCBFW1VuaXggdGltZXN0YW1wXVxuICAgIEQgLS0-fHJlZGlzIGtleXwgR1tyZWRpcy1zZXJ2ZXJdXG4gICAgRSAtLT58cmVkaXMgdmFyfCBHXG4gICAgRiAtLT58cmVkaXMgdHRsfCBHIiwibWVybWFpZCI6eyJ0aGVtZSI6ImRlZmF1bHQifSwidXBkYXRlRWRpdG9yIjpmYWxzZSwiYXV0b1N5bmMiOnRydWUsInVwZGF0ZURpYWdyYW0iOmZhbHNlfQ)](https://mermaid-js.github.io/mermaid-live-editor/edit/###eyJjb2RlIjoiZ3JhcGggVERcbiAgICBBW0VyciBJUCBMaXN0IEpvYiBdIC0tPiBCKENyZWF0ZSBJQ01QIFBhY2thZ2UpXG4gICAgQ1tkZWZpbml0ZSB0aW1lIElQIExpc3QgSm9iIDEwcyBdIC0tPiBCKENyZWF0ZSBJQ01QIFBhY2thZ2UpXG4gICAgRFtkZWZpbml0ZSB0aW1lIElQIExpc3QgSm9iIDYwcyBdIC0tPiBCKENyZWF0ZSBJQ01QIFBhY2thZ2UpXG4gICAgQiAtLT4gfGJvZHkgdW5pbnggdGltZXN0YW1wfEVbSUNNUCBQYWNrYWdlXVxuICAgIEUgLS0-IEZbYV1cblxuICAiLCJtZXJtYWlkIjoie1xuICBcInRoZW1lXCI6IFwiZGVmYXVsdFwiXG59IiwidXBkYXRlRWRpdG9yIjpmYWxzZSwiYXV0b1N5bmMiOnRydWUsInVwZGF0ZURpYWdyYW0iOmZhbHNlfQ)

## agent jobs
[![](https://mermaid.ink/img/eyJjb2RlIjoiZ3JhcGggVERcbiAgICBBW0VyciBJUCBMaXN0IEpvYiBdIC0tPiBCKENyZWF0ZSBJQ01QIFBhY2thZ2UpXG4gICAgQ1tkZWZpbml0ZSB0aW1lIElQIExpc3QgSm9iIDEwcyBdIC0tPiBCKENyZWF0ZSBJQ01QIFBhY2thZ2UpXG4gICAgRFtkZWZpbml0ZSB0aW1lIElQIExpc3QgSm9iIDYwcyBdIC0tPiBCKENyZWF0ZSBJQ01QIFBhY2thZ2UpXG4gICAgQiAtLT4gfGJvZHkgdW5pbnggdGltZXN0YW1wfEVbSUNNUCBQYWNrYWdlXVxuICAgIEUgLS0-IHxzZW5kfEZbVHJhZ2VudCBJUF1cblxuICAiLCJtZXJtYWlkIjp7InRoZW1lIjoiZGVmYXVsdCJ9LCJ1cGRhdGVFZGl0b3IiOmZhbHNlLCJhdXRvU3luYyI6dHJ1ZSwidXBkYXRlRGlhZ3JhbSI6ZmFsc2V9)](https://mermaid-js.github.io/mermaid-live-editor/edit/##eyJjb2RlIjoiZ3JhcGggVERcbiAgICBBW0VyciBJUCBMaXN0IEpvYiBdIC0tPiBCKENyZWF0ZSBJQ01QIFBhY2thZ2UpXG4gICAgQ1tkZWZpbml0ZSB0aW1lIElQIExpc3QgSm9iIDEwcyBdIC0tPiBCKENyZWF0ZSBJQ01QIFBhY2thZ2UpXG4gICAgRFtkZWZpbml0ZSB0aW1lIElQIExpc3QgSm9iIDYwcyBdIC0tPiBCKENyZWF0ZSBJQ01QIFBhY2thZ2UpXG4gICAgQiAtLT4gfGJvZHkgdW5pbnggdGltZXN0YW1wfEVbSUNNUCBQYWNrYWdlXVxuICAgIEUgLS0-IHxzZW5kfCBGW1RyYWdlbnQgSVBdXG5cbiAgIiwibWVybWFpZCI6IntcbiAgXCJ0aGVtZVwiOiBcImRlZmF1bHRcIlxufSIsInVwZGF0ZUVkaXRvciI6ZmFsc2UsImF1dG9TeW5jIjp0cnVlLCJ1cGRhdGVEaWFncmFtIjpmYWxzZX0)


## update err IP list

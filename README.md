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
1. 程序根据 cron job 中的 ip list 发送 icmp 包, 包中包含发送时的 unixnano
2. 对端返回 icmp pack ，将包中的 unixnano 与 time.Now() 相减获得 ping值
3. 将 ping值 写入 redis ,并设置 timeout。
4. 如 redis key timeout，agent 将该Ip加入 err list 
5. cron job 定时检查 memory 中的 ip status，如无收到包 或 ping timeout，将IP 加入 err list
6. 每分钟检查 err list 掉包率,如低于检查值从 err list 移走 ip

## redis key

```bash
// 所有 agent 列表，agent 自写入
agent-list, set

// agent 在线状态
agent-online_$ageent, key & timeout

// ping is err list 表, 定时检查 ip 时间段内是否有丢包，如无则移出。
agent-err-list_$agent, set
["xx", "xx",  "xx"]

// cron job list 名称
agent-jobs_$agent,  set 
val: ["xxx", "kkk", "ccc"]

// job status
agent_$agent_$jobname, key
{SPEC:"* * * * *" , name:"xxxx", group:["group1", "ccc"]}

// ip group list
key: groups-list, set
val: ["group1","ccc","bbb"]

// ip group 
key: group_$groupName, set
val: ["192.18.1.1", "192.168.2.1"]

// ip 最新 ms 值 , agent 监听 timeout 事件，如发生将 ip ，添加至 err list
key: $agent_$ip,  key
ms

// ip 状态 
key: $agent_all-ip-status,  key
{ip: {SendCount, ReturnCount, ip, InErrList} }

// agent 的错误IP 列表
key: $agent_err_ip, set
val: ["192.16.111.1"]


```


## agent Listen ICMP package
[![](https://mermaid.ink/img/eyJjb2RlIjoiZ3JhcGggVERcbiAgICBBW0xpc3RlbiBJQ01QIFNlcnZlcl0gLS0-fGdldCBpY21wIHBhY2thZ2V8IEIocmVhZCBwYWNrYWdlIGpvYilcbiAgICBCIC0tPnxtZW1vcnkgbWFwfCBGW2dldCBpcCB0dGxdXG4gICAgQiAtLT58aXAgaGVhZHwgRFtzcmMgaXBdXG4gICAgQiAtLT58cGFja2FnZSBib2R5fCBFW1VuaXggdGltZXN0YW1wXVxuICAgIEQgLS0-fHJlZGlzIGtleXwgR1tyZWRpcy1zZXJ2ZXJdXG4gICAgRSAtLT58cmVkaXMgdmFyfCBHXG4gICAgRiAtLT58cmVkaXMgdHRsfCBHIiwibWVybWFpZCI6eyJ0aGVtZSI6ImRlZmF1bHQifSwidXBkYXRlRWRpdG9yIjpmYWxzZSwiYXV0b1N5bmMiOnRydWUsInVwZGF0ZURpYWdyYW0iOmZhbHNlfQ)](https://mermaid-js.github.io/mermaid-live-editor/edit/###eyJjb2RlIjoiZ3JhcGggVERcbiAgICBBW0VyciBJUCBMaXN0IEpvYiBdIC0tPiBCKENyZWF0ZSBJQ01QIFBhY2thZ2UpXG4gICAgQ1tkZWZpbml0ZSB0aW1lIElQIExpc3QgSm9iIDEwcyBdIC0tPiBCKENyZWF0ZSBJQ01QIFBhY2thZ2UpXG4gICAgRFtkZWZpbml0ZSB0aW1lIElQIExpc3QgSm9iIDYwcyBdIC0tPiBCKENyZWF0ZSBJQ01QIFBhY2thZ2UpXG4gICAgQiAtLT4gfGJvZHkgdW5pbnggdGltZXN0YW1wfEVbSUNNUCBQYWNrYWdlXVxuICAgIEUgLS0-IEZbYV1cblxuICAiLCJtZXJtYWlkIjoie1xuICBcInRoZW1lXCI6IFwiZGVmYXVsdFwiXG59IiwidXBkYXRlRWRpdG9yIjpmYWxzZSwiYXV0b1N5bmMiOnRydWUsInVwZGF0ZURpYWdyYW0iOmZhbHNlfQ)

## agent jobs
[![](https://mermaid.ink/img/eyJjb2RlIjoiZ3JhcGggVERcbiAgICBBW0VyciBJUCBMaXN0IEpvYiBdIC0tPiBCKENyZWF0ZSBJQ01QIFBhY2thZ2UpXG4gICAgQ1tkZWZpbml0ZSB0aW1lIElQIExpc3QgSm9iIDEwcyBdIC0tPiBCKENyZWF0ZSBJQ01QIFBhY2thZ2UpXG4gICAgRFtkZWZpbml0ZSB0aW1lIElQIExpc3QgSm9iIDYwcyBdIC0tPiBCKENyZWF0ZSBJQ01QIFBhY2thZ2UpXG4gICAgQiAtLT4gfGJvZHkgdW5pbnggdGltZXN0YW1wfEVbSUNNUCBQYWNrYWdlXVxuICAgIEUgLS0-IHxzZW5kfEZbVHJhZ2VudCBJUF1cblxuICAiLCJtZXJtYWlkIjp7InRoZW1lIjoiZGVmYXVsdCJ9LCJ1cGRhdGVFZGl0b3IiOmZhbHNlLCJhdXRvU3luYyI6dHJ1ZSwidXBkYXRlRGlhZ3JhbSI6ZmFsc2V9)](https://mermaid-js.github.io/mermaid-live-editor/edit/##eyJjb2RlIjoiZ3JhcGggVERcbiAgICBBW0VyciBJUCBMaXN0IEpvYiBdIC0tPiBCKENyZWF0ZSBJQ01QIFBhY2thZ2UpXG4gICAgQ1tkZWZpbml0ZSB0aW1lIElQIExpc3QgSm9iIDEwcyBdIC0tPiBCKENyZWF0ZSBJQ01QIFBhY2thZ2UpXG4gICAgRFtkZWZpbml0ZSB0aW1lIElQIExpc3QgSm9iIDYwcyBdIC0tPiBCKENyZWF0ZSBJQ01QIFBhY2thZ2UpXG4gICAgQiAtLT4gfGJvZHkgdW5pbnggdGltZXN0YW1wfEVbSUNNUCBQYWNrYWdlXVxuICAgIEUgLS0-IHxzZW5kfCBGW1RyYWdlbnQgSVBdXG5cbiAgIiwibWVybWFpZCI6IntcbiAgXCJ0aGVtZVwiOiBcImRlZmF1bHRcIlxufSIsInVwZGF0ZUVkaXRvciI6ZmFsc2UsImF1dG9TeW5jIjp0cnVlLCJ1cGRhdGVEaWFncmFtIjpmYWxzZX0)


## update err IP list
[![](https://mermaid.ink/img/eyJjb2RlIjoiZ3JhcGggVERcbiAgICBBW0xpc3RlbiBJQ01QIFNlcnZlcl0gLS0-fGdldCBpY21wIHBhY2thZ2V8IEIocmVhZCBwYWNrYWdlIGpvYilcbiAgICBCIC0tPnxtZW1vcnkgbWFwfCBGW2dldCBpcCB0dGxdXG4gICAgQiAtLT58aXAgaGVhZHwgRFtzcmMgaXBdXG4gICAgQiAtLT58cGFja2FnZSBib2R5fCBFW1VuaXggdGltZXN0YW1wXVxuICAgIEQgLS0-fHJlZGlzIGtleXwgR1tyZWRpcy1zZXJ2ZXJdXG4gICAgRSAtLT58cmVkaXMgdmFyfCBHXG4gICAgRiAtLT58cmVkaXMgdHRsfCBHIiwibWVybWFpZCI6eyJ0aGVtZSI6ImRlZmF1bHQifSwidXBkYXRlRWRpdG9yIjpmYWxzZSwiYXV0b1N5bmMiOnRydWUsInVwZGF0ZURpYWdyYW0iOmZhbHNlfQ)](https://mermaid-js.github.io/mermaid-live-editor/edit/###eyJjb2RlIjoiZ3JhcGggVERcbiAgICBBW0VyciBJUCBMaXN0IEpvYiBdIC0tPiBCKENyZWF0ZSBJQ01QIFBhY2thZ2UpXG4gICAgQ1tkZWZpbml0ZSB0aW1lIElQIExpc3QgSm9iIDEwcyBdIC0tPiBCKENyZWF0ZSBJQ01QIFBhY2thZ2UpXG4gICAgRFtkZWZpbml0ZSB0aW1lIElQIExpc3QgSm9iIDYwcyBdIC0tPiBCKENyZWF0ZSBJQ01QIFBhY2thZ2UpXG4gICAgQiAtLT4gfGJvZHkgdW5pbnggdGltZXN0YW1wfEVbSUNNUCBQYWNrYWdlXVxuICAgIEUgLS0-IEZbYV1cblxuICAiLCJtZXJtYWlkIjoie1xuICBcInRoZW1lXCI6IFwiZGVmYXVsdFwiXG59IiwidXBkYXRlRWRpdG9yIjpmYWxzZSwiYXV0b1N5bmMiOnRydWUsInVwZGF0ZURpYWdyYW0iOmZhbHNlfQ)
hello-abc  受限中继代理（Restricted Relay Proxy）
----

[TOC]

## 功能
角色B 主动通过tcp协议连接上 A 和 C  

使用者和A的端口通信，通过转发链 A-->B-->C ，通信数据达到C

限制条件：
1. 使用者只能连上A
2. B可以单向连上A和C， 反之不行

## 使用

注意⚠ ：整个隧道同时只支持一个用户使用

在程序同一个目录下，新建配置文件 `config.toml`

内容为:
```toml
# 角色： a/b/c
role = "a"

[a]
ip = "127.0.0.1"
port_tunnel = 8080
port_server = 8081

[c]
ip = "127.0.0.1"
port_tunnel = 8083
port_forward = 22
```

三个角色可以复用同一个配置文件，这样简单、不容易出错

下面给出每个角色工作时最小的配置文件

### a


a 监听隧道端口，监听 server 端口，再把 server端口上的数据 转发到隧道端口上

```toml
role = "a"

[a]
port_tunnel = 8080
port_server = 8081
```

### b

b需要去连接a、c的隧道端口，其他不需要

```toml
role = "b"

[a]
ip = "127.0.0.1"
port_tunnel = 8080

[c]
ip = "127.0.0.1"
port_tunnel = 8083
```

### c
c 监听隧道端口，转发到 forward 端口上

```toml
role = "c"


[c]
port_tunnel = 8083
port_forward = 22
```


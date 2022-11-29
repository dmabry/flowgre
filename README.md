# Flowgre
Slinging packets since 2022!


```
    ___ _                             
  / __\ | _____      ____ _ _ __ ___
 / _\ | |/ _ \ \ /\ / / _` | '__/ _ \
/ /   | | (_) \ V  V / (_| | | |  __/
\/    |_|\___/ \_/\_/ \__, |_|  \___|
                      |___/
```
For sending fabricated Netflow v9 traffic at a collector for testing

[![Build Status](https://drone.dmabry.net/api/badges/dmabry/flowgre/status.svg?ref=refs/heads/main)](https://drone.dmabry.net/dmabry/flowgre)
[![Go Report Card](https://goreportcard.com/badge/github.com/dmabry/flowgre)](https://goreportcard.com/report/github.com/dmabry/flowgre)
[![Go Reference](https://pkg.go.dev/badge/github.com/dmabry/flowgre.svg)](https://pkg.go.dev/github.com/dmabry/flowgre)
## Single
```shell
Single is used to send a given number of flows in sequence to a collector for testing.
Right now, Source and Destination IPs are randomly generated in the 10.0.0.0/8 range and hardcoded for HTTPS flows.

Usage of flowgre single:

  -count int
    	count of flow to send in sequence. (default 1)
  -hexdump
    	If true, do a hexdump of the packet
  -port int
    	destination port used by the flow collector. (default 9995)
  -server string
    	servername or ip address of flow collector. (default "localhost")
  -srcport int
    	source port used by the client. If 0 a Random port between 10000-15000
```

### Example Use
```shell
flowgre single -server 10.10.10.10 -count 10
```

## Barrage
```shell
Barrage is used to send a continuous barrage of flows in different sequence to a collector for testing.

Usage of flowgre barrage:

  -config string
    	Config file to use.  Supersedes all given args
  -delay int
    	number of milliseconds between packets sent (default 100)
  -port int
    	destination port used by the flow collector (default 9995)
  -server string
    	servername or ip address of the flow collector (default "127.0.0.1")
  -web
    	Whether to use the web server or not
  -web-ip string
    	IP address the web server will listen on (default "0.0.0.0")
  -web-port int
    	Port to bind the web server on (default 8080)
  -workers int
    	number of workers to create. Unique sources per worker (default 4)
```

## Example Config File
```yaml
targets:
  server1:
    ip: 127.0.0.1
    port: 9995
    workers: 4
    delay: 100
```

## Record
```shell
Record is used to record flows to a file for later replay testing.

Usage of flowgre record:

  -db string
        Directory to place recorded flows for later replay (default "recorded_flows")
  -ip string
        ip address record should listen on (default "127.0.0.1")
  -port int
        listen udp port (default 9995)
  -verbose
        Whether to log every packet received. Warning can be a lot
```

## Replay
```shell
Replay is used to send recorded flows to a target server.

Usage of flowgre replay:

  -db string
        Directory to read recorded flows from (default "recorded_flows")
  -delay int
        number of milliseconds between packets sent (default 100)
  -loop
        Loops the replays forever
  -port int
        target server udp port (default 9995)
  -server string
        target server to replay flows at (default "127.0.0.1")
  -verbose
        Whether to log every packet received. Warning can be a lot
  -workers int
        Number of workers to spawn for replay (default 1)
```

## Proxy
```shell
Proxy is used to accept flows and relay them to multiple targets

Usage of flowgre proxy:

  -ip string
    	ip address proxy should listen on (default "127.0.0.1")
  -port int
    	proxy listen udp port (default 9995)
  -target value
    	Can be passed multiple times in IP:PORT format
  -verbose
    	Whether to log every flow received. Warning can be a lot
```

## Web Dashboard
Flowgre provides a basic web dashboard that will display the number of workers, how much work they've done and the
config used to start Flowgre.  The stats shown all come from the stats collector and should match the stdout worker
stats.

![Dashboard Image](https://github.com/dmabry/flowgre/blob/main/docs/images/dashboard.png?raw=true)

## License
Licensed to the Flowgre Team under one
or more contributor license agreements. The Flowgre Team licenses this file to you
under the Apache License, Version 2.0 (the "License"); 
you may not use this file except in compliance
with the License.  You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing,
software distributed under the License is distributed on an
"AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
KIND, either express or implied.  See the License for the
specific language governing permissions and limitations
under the License.

Please see the [LICENSE](LICENSE) file included in the root directory
of the source tree for extended license details.

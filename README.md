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
  -workers int
    	number of workers to create. Unique sources per worker (default 4)
```

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

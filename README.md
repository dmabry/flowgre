# Flowgre
Slinging packets since 2022!


```shell   
    ___ _                             
  / __\ | _____      ____ _ _ __ ___
 / _\ | |/ _ \ \ /\ / / _` | '__/ _ \
/ /   | | (_) \ V  V / (_| | | |  __/
\/    |_|\___/ \_/\_/ \__, |_|  \___|
                      |___/
```

## Single
```shell
Single is used to send a given number of flows in sequence to a collector for testing.
Right now, Source and Destination IPs are randomly generated in the 10.0.0.0/8 range.

Usage of ./flowgre:

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
COMING SOON!
Barrage is used to send a continuous barrage of flows in different sequence to a collector for testing.
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
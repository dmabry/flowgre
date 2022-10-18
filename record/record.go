// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

// Barrage is used to setup a continuous stream of netflow packets towards a single collector

package record

import (
	"log"
	"os"
)

// Run Record
func Run(ip string, port int, file string) {
	log.Printf("Listening on %s:%d recording to %s.", ip, port, file)
	os.Exit(0)
}

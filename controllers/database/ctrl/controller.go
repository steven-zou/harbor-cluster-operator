// Copyright Project Harbor Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ctrl

import (
	"github.com/zalando/postgres-operator/pkg/util/k8sutil"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/zalando/postgres-operator/pkg/controller"
	"github.com/zalando/postgres-operator/pkg/spec"
)

var config spec.ControllerConfig

func mustParseDuration(d string) time.Duration {
	duration, err := time.ParseDuration(d)
	if err != nil {
		panic(err)
	}
	return duration
}

func init() {
	configMapRawName := os.Getenv("CONFIG_MAP_NAME")
	if configMapRawName != "" {

		err := config.ConfigMapName.Decode(configMapRawName)
		if err != nil {
			log.Fatalf("incorrect config map name: %v", configMapRawName)
		}

		log.Printf("Fully qualified configmap name: %v", config.ConfigMapName)

	}
	if crdInterval := os.Getenv("CRD_READY_WAIT_INTERVAL"); crdInterval != "" {
		config.CRDReadyWaitInterval = mustParseDuration(crdInterval)
	} else {
		config.CRDReadyWaitInterval = 4 * time.Second
	}

	if crdTimeout := os.Getenv("CRD_READY_WAIT_TIMEOUT"); crdTimeout != "" {
		config.CRDReadyWaitTimeout = mustParseDuration(crdTimeout)
	} else {
		config.CRDReadyWaitTimeout = 30 * time.Second
	}
}

// Start the psql operator controller
func Start() {
	var err error

	log.SetOutput(os.Stdout)
	log.Printf("Spilo operator %s\n", "1.5.0")

	sigs := make(chan os.Signal, 1)
	stop := make(chan struct{})
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM) // Push signals into channel

	wg := &sync.WaitGroup{} // Goroutines can add themselves to this to be waited on

	config.RestConfig, err = k8sutil.RestConfig("", false)
	if err != nil {
		log.Fatalf("couldn't get REST config: %v", err)
	}

	c := controller.NewController(&config, "")

	c.Run(stop, wg)

	sig := <-sigs
	log.Printf("Shutting down... %+v", sig)

	close(stop) // Tell goroutines to stop themselves
	wg.Wait()   // Wait for all to be stopped
}

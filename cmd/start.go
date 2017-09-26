// Copyright © 2017 NAME HERE <EMAIL ADDRESS>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime/pprof"
	"strings"
	"sync"
	"syscall"

	"github.com/Sirupsen/logrus"
	"github.com/jdamick/tup/backend"
	"github.com/jdamick/tup/config"
	"github.com/jdamick/tup/udp"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// startCmd represents the start command
var startCmd = &cobra.Command{
	Use:   "start",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		// TODO: Work your own magic here
		fmt.Println("start called")

		cpuprofile := viper.GetString("cpuprofile")
		if len(cpuprofile) > 0 {
			f, err := os.Create(cpuprofile)
			if err != nil {
				log.Fatal(err)
			}
			pprof.StartCPUProfile(f)
			defer pprof.StopCPUProfile()
		}

		logrus.SetLevel(logrus.InfoLevel)

		if viper.GetBool("debug") {
			logrus.SetLevel(logrus.DebugLevel)
		}

		c := config.DefaultConfig()

		for _, addr := range backends() {
			c.Backends = append(c.Backends, config.Backend{Addr: addr})
		}

		c.ProxyAddr = "0.0.0.0:9090"
		b := backend.NewManager(c)
		udpProxy := udp.NewUDPProxy(c)
		udpProxy.BackendManager = b
		var wg sync.WaitGroup

		shutdownHandler(&wg)
		wg.Add(1)
		udpProxy.Start()
		wg.Wait()
	},
}

func shutdownHandler(wg *sync.WaitGroup) {
	var signals = []os.Signal{syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGABRT}
	ch := make(chan os.Signal, 2)
	signal.Notify(ch, signals...)

	go func() {
		select {
		case s := <-ch:
			_ = s
			//log.Debugf("received signal: %s", s)
			wg.Done()
		}
	}()
}

func backends() []string {
	backends := viper.GetString("backends")
	list := strings.Split(backends, ",")
	for i, v := range list {
		list[i] = strings.TrimSpace(v)
	}
	return list
}

func init() {

	startCmd.PersistentFlags().StringP("cpuprofile", "c", "", "cpu profile file")
	viper.BindPFlag("cpuprofile", startCmd.PersistentFlags().Lookup("cpuprofile"))

	startCmd.PersistentFlags().StringP("backends", "b", "", "Comma-delimited list of backend addresses (ip:port)")
	viper.BindPFlag("backends", startCmd.PersistentFlags().Lookup("backends"))
	RootCmd.AddCommand(startCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// startCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// startCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

}

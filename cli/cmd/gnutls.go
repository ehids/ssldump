//go:build !androidgki
// +build !androidgki

// Copyright 2022 CFC4N <cfc4n.cs@gmail.com>. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"context"
	"github.com/gojue/ecapture/pkg/util/kernel"
	"github.com/gojue/ecapture/user/config"
	"github.com/gojue/ecapture/user/module"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/spf13/cobra"
)

var gc = config.NewGnutlsConfig()

// gnutlsCmd represents the openssl command
var gnutlsCmd = &cobra.Command{
	Use:     "gnutls",
	Aliases: []string{"gnu"},
	Short:   "capture gnutls text content without CA cert for gnutls libraries.",
	Long: `use eBPF uprobe/TC to capture process event data.
ecapture gnutls
ecapture gnutls --hex --pid=3423
ecapture gnutls -l save.log --pid=3423
ecapture gnutls --gnutls=/lib/x86_64-linux-gnu/libgnutls.so
`,
	Run: gnuTlsCommandFunc,
}

func init() {
	//opensslCmd.PersistentFlags().StringVar(&gc.Curlpath, "wget", "", "wget file path, default: /usr/bin/wget. (Deprecated)")
	gnutlsCmd.PersistentFlags().StringVar(&gc.Gnutls, "gnutls", "", "libgnutls.so file path, will automatically find it from curl default.")
	rootCmd.AddCommand(gnutlsCmd)
}

// gnuTlsCommandFunc executes the "bash" command.
func gnuTlsCommandFunc(command *cobra.Command, args []string) {
	stopper := make(chan os.Signal, 1)
	signal.Notify(stopper, os.Interrupt, syscall.SIGTERM)
	ctx, cancelFun := context.WithCancel(context.TODO())

	logger := log.New(os.Stdout, "tls_", log.LstdFlags)

	// save global config
	gConf, err := getGlobalConf(command)
	if err != nil {
		logger.Fatal(err)
	}
	logger.SetOutput(gConf.writer)

	logger.Printf("ECAPTURE :: %s Version : %s", cliName, GitVersion)
	logger.Printf("ECAPTURE :: Pid Info : %d", os.Getpid())
	var version kernel.Version
	version, _ = kernel.HostVersion() // it's safe to ignore error because we have checked it in func main
	logger.Printf("ECAPTURE :: Kernel Info : %s", version.String())
	modNames := []string{module.ModuleNameGnutls}

	var runMods uint8
	var runModules = make(map[string]module.IModule)
	var wg sync.WaitGroup

	for _, modName := range modNames {
		mod := module.GetModuleByName(modName)
		if mod == nil {
			logger.Printf("ECAPTURE :: \tcant found module: %s", modName)
			break
		}
		if gc == nil {
			logger.Printf("ECAPTURE :: \tcant found module %s config info.", mod.Name())
			break
		}
		var conf config.IConfig
		conf = gc

		conf.SetPid(gConf.Pid)
		conf.SetUid(gConf.Uid)
		conf.SetDebug(gConf.Debug)
		conf.SetHex(gConf.IsHex)
		conf.SetPerCpuMapSize(gConf.mapSizeKB)

		err = conf.Check()

		if err != nil {
			logger.Printf("%s\tmodule initialization failed. [skip it]. error:%+v", mod.Name(), err)
			continue
		}

		logger.Printf("%s\tmodule initialization", mod.Name())

		//初始化
		err = mod.Init(ctx, logger, conf)
		if err != nil {
			logger.Printf("%s\tmodule initialization failed, [skip it]. error:%+v", mod.Name(), err)
			continue
		}

		err = mod.Run()
		if err != nil {
			logger.Printf("%s\tmodule run failed, [skip it]. error:%+v", mod.Name(), err)
			continue
		}
		runModules[mod.Name()] = mod
		logger.Printf("%s\tmodule started successfully.", mod.Name())
		wg.Add(1)
		runMods++
	}

	// needs runmods > 0
	if runMods > 0 {
		logger.Printf("ECAPTURE :: \tstart %d modules", runMods)
		<-stopper
	} else {
		logger.Println("ECAPTURE :: \tNo runnable modules, Exit(1)")
		os.Exit(1)
	}
	cancelFun()

	// clean up
	for _, mod := range runModules {
		err = mod.Close()
		wg.Done()
		if err != nil {
			logger.Fatalf("%s\tmodule close failed. error:%+v", mod.Name(), err)
		}
	}

	wg.Wait()
	os.Exit(0)
}

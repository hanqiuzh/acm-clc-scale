/*
Copyright © 2020 NAME HERE <EMAIL ADDRESS>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"fmt"
	"sync"
	"time"

	"github.com/hanqiuzh/acm-clc-scale/mockcluster/pkg/utils"
	"github.com/spf13/cobra"
)

var sleepSeconds = 10
var keepAlive = true
var total = 1
var wg sync.WaitGroup

// addCmd represents the add command
var addCmd = &cobra.Command{
	Use:   "add total",
	Short: "add mock clusters",
	Long:  `Add mock clusters. Will set cluster online with keep-alive enabled.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("add called to add %d clusters\n", total)
		if total <= 0 || numConcurrent <= 0 || sleepSeconds <= 0 {
			fmt.Println("total & numConcurrent & sleepSconds should be larger than 0")
			return
		}
		clusterDynamicClient := createDynamicClient()
		m := utils.MockClusterClient{
			Client: clusterDynamicClient,
			Prefix: prefix,
		}

		if keepAlive {
			fmt.Println("keep-alive enabled, will keep cluster alive")
			wg.Add(1)
			go func() {
				m.KeepAliveAll(numConcurrent)
			}()
		}

		numWait := numConcurrent
		if total < numWait {
			numWait = total
		}
		midName := fmt.Sprintf("%d", time.Now().UnixNano()/int64(time.Second))

		d := total / numWait
		r := total - d*numWait
		fmt.Printf("will have %d threads\n", numWait)
		for i := 0; i < numWait; i++ {
			itemNum := d
			if r > 0 {
				r--
				itemNum++
			}
			wg.Add(1)
			go func(id int) {
				//fmt.Printf("%d branch: will create %d clusters\n", id, itemNum)
				for j := 0; j < itemNum; j++ {
					name := fmt.Sprintf("%s%s-%d-%d", prefix, midName, id, j)
					fmt.Println("adding " + name)
					err := m.CreateMockManagedCluster(name)
					if err != nil {
						fmt.Printf("error adding %d: %v\n", j, err)
					}
					time.Sleep(time.Second * time.Duration(sleepSeconds))
				}
				wg.Done()
			}(i)
		}
		wg.Wait()
	},
}

func init() {
	rootCmd.AddCommand(addCmd)
	// Here you will define your flags and configuration settings.
	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// addCmd.PersistentFlags().String("foo", "", "A help for foo")
	addCmd.PersistentFlags().IntVarP(&sleepSeconds, "sleep", "s", 10, "seconds to sleep after fire a group of concurrent add cluster requests.")
	addCmd.PersistentFlags().IntVarP(&total, "total", "t", 1, "total number of clusters to add.")
	addCmd.PersistentFlags().BoolVarP(&keepAlive, "keep-alive", "k", true, "if set true, will update lease for all added mock clusters")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// addCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

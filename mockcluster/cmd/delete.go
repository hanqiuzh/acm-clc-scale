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
	"time"

	"github.com/hanqiuzh/acm-clc-scale/mockcluster/pkg/utils"
	"github.com/spf13/cobra"
)

var deleteSleepSeconds int

// deleteCmd represents the delete command
var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete mock clusters.",
	Long:  `Delete mock clusters.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("delete called to delete %d clusters\n", totalDelete)
		if totalDelete <= 0 || numConcurrent <= 0 || deleteSleepSeconds <= 0 {
			fmt.Println("totalDelete & numConcurrent & sleepSconds should be larger than 0")
			return
		}

		clusterDynamicClient := createDynamicClient()
		m := utils.MockClusterClient{
			Client: clusterDynamicClient,
			Prefix: prefix,
		}
		l, err := m.ListMockClusters()
		if err != nil {
			fmt.Printf("error %s\n", err)
			return
		}

		if totalDelete > len(l) {
			totalDelete = len(l)
		}
		if totalDelete == 0 {
			return
		}
		numWait := numConcurrent
		if totalDelete < numWait {
			numWait = totalDelete
		}
		d := totalDelete / numWait
		r := totalDelete - d*numWait
		fmt.Printf("will have %d threads\n", numWait)
		idx := 0
		for i := 0; i < numWait; i++ {
			itemNum := d
			if r > 0 {
				r--
				itemNum++
			}
			clusters := make([]string, 0)
			for j := 0; j < itemNum; j++ {
				if idx < len(l) {
					clusters = append(clusters, l[idx])
					idx++
				}
			}

			wg.Add(1)
			go func() {
				for _, cluster := range clusters {
					fmt.Printf("deleting %s\n", cluster)
					if err := m.DetachMockManagedCluster(cluster); err != nil {
						fmt.Printf("error deleting %s: %v\n", cluster, err)
					}
					time.Sleep(time.Second * time.Duration(deleteSleepSeconds))
				}
				//
				wg.Done()
			}()
		}
		wg.Wait()
	},
}

var totalDelete int

func init() {
	rootCmd.AddCommand(deleteCmd)

	// Here you will define your flags and configuration settings.
	deleteCmd.PersistentFlags().IntVarP(&deleteSleepSeconds, "sleep", "s", 1, "seconds to sleep after fire a group of concurrent add cluster requests.")
	deleteCmd.PersistentFlags().IntVarP(&totalDelete, "total", "t", 1, "total number of clusters to delete.")
	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// deleteCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// deleteCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

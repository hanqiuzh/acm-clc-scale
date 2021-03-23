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
	"os"
	"path/filepath"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var kubeconfig string
var prefix = "mock-cluster-"
var numConcurrent = 1
var qps float32 = 100.
//hubCfg hold the input rest.Config, which will be used to retrive masterUrl
var hubCfg *rest.Config

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "mockcluster",
	Short: "A tool of mocking clusters",
	Long:  `A tool of mocking clusters. Clusters will have the label 'mock-cluster=true'.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	//	Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().StringVar(&kubeconfig, "kubeconfig", getDefaultKubeconfig(), "(optional) absolute path to the kubeconfig file")
	rootCmd.PersistentFlags().StringVarP(&prefix, "prefix", "p", "mock-cluster-", "prefix of cluster name. Naming convension will be prefixRandomNumber")
	rootCmd.PersistentFlags().IntVarP(&numConcurrent, "concurrent", "c", 1, "number of concurrent actions. will be applied for add/delete/keep-alive")
	rootCmd.PersistentFlags().Float32Var(&qps, "qps", 200, "kube client qps")
	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {

}

func getDefaultKubeconfig() string {
	kubeconfig := ""
	if home, _ := homedir.Dir(); home != "" {
		kubeconfig = filepath.Join(home, ".kube", "config")
	}
	if os.Getenv("KUBECONFIG") != "" {
		kubeconfig = os.Getenv("KUBECONFIG")
	}
	return kubeconfig
}

func createDynamicClient() dynamic.Interface {
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		panic(err)
	}
	config.QPS = qps
	client, err := dynamic.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	hubCfg = config
	return client
}

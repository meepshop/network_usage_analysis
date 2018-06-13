// Copyright © 2018 NAME HERE <EMAIL ADDRESS>
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
	"log"

	"github.com/meepshop/network_usage_analysis/record"
	"github.com/spf13/cobra"
)

// analysisCmd represents the analysis command
var analysisCmd = &cobra.Command{
	Use:   "analysis",
	Short: "從storage讀取流量資料，分析整理後塞入db",
	Long:  `從storage讀取流量資料，分析整理後塞入db，以小時為單位，每次執行會從上次停止處接續處理`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Println("Analysis Start")
		record.Analysis()
		log.Println("Analysis Done")
	},
}

func init() {
	rootCmd.AddCommand(analysisCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// analysisCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// analysisCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

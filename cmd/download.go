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
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"

	"cloud.google.com/go/storage"
	"github.com/spf13/cobra"
	"google.golang.org/api/iterator"
)

// downloadCmd represents the download command
var downloadCmd = &cobra.Command{
	Use:   "download",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		client, err := storage.NewClient(ctx)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		defer client.Close()
		bucket := client.Bucket("network_usage")
		objs := bucket.Objects(ctx, &storage.Query{
			Prefix: "requests/2018/05",
		})
		for {
			attrs, err := objs.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			matched, err := regexp.MatchString(".*S0.json", attrs.Name)
			if err != nil {
				fmt.Println(err)
				continue
			}
			if matched {
				fmt.Println(attrs.MediaLink)
			}
			resp, err := http.Get(attrs.MediaLink)
			if err != nil {
				fmt.Println(err)
				continue
			}
			file, _ := os.OpenFile(attrs.Name, os.O_RDONLY|os.O_CREATE, 0666)
			result, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				fmt.Println(err)
				continue
			}
			fmt.Println(string(result))
			n, err := file.Write(result)
			if err != nil {
				fmt.Println("file write", err)
				continue
			}
			fmt.Println(n, "writed")
		}
		fmt.Println("download called")
	},
}

func init() {
	rootCmd.AddCommand(downloadCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// downloadCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// downloadCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

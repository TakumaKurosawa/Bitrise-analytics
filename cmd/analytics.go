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
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"time"

	"github.com/spf13/cobra"
)

type Date struct {
	BeforeDate time.Time
	AfterDate  time.Time
}

type Response struct {
	Data   []Data `json:"data"`
	Paging Paging `json:"paging"`
}

type Data struct {
	StartedOnWorkerAt time.Time `json:"started_on_worker_at"`
	FinishedAt        time.Time `json:"finished_at"`
	Status            int       `json:"status"`
}

type Paging struct {
	TotalItemCount int    `json:"total_item_count"`
	PageItemLimit  int    `json:"page_item_limit"`
	Next           string `json:"next"`
}

// monthlyCmd represents the get command
var monthlyCmd = &cobra.Command{
	Use:   "analytics",
	Short: "ビルドアナリティクス",
	Long: `【ビルドアナリティクス】
・ビルド回数
・ビルド合計時間
・ビルド1回あたりの平均時間
を算出します。`,
	Run: func(cmd *cobra.Command, args []string) {
		now := time.Now()

		var thisMonth Date
		thisMonth.AfterDate = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		thisMonth.BeforeDate = thisMonth.AfterDate.AddDate(0, 1, -1).Add(23 * time.Hour).Add(59 * time.Minute).Add(59 * time.Second)

		var lastMonth Date
		lastMonth.AfterDate = thisMonth.AfterDate.AddDate(0, -1, 0)
		lastMonth.BeforeDate = lastMonth.AfterDate.AddDate(0, 1, -1).Add(23 * time.Hour).Add(59 * time.Minute).Add(59 * time.Second)

		var twoMonthBefore Date
		twoMonthBefore.AfterDate = thisMonth.AfterDate.AddDate(0, -2, 0)
		twoMonthBefore.BeforeDate = twoMonthBefore.AfterDate.AddDate(0, 1, -1).Add(23 * time.Hour).Add(59 * time.Minute).Add(59 * time.Second)

		thisMonthData := sendAPIRequest(thisMonth.BeforeDate, thisMonth.AfterDate)
		lastMonthData := sendAPIRequest(lastMonth.BeforeDate, lastMonth.AfterDate)
		twoMonthBeforeData := sendAPIRequest(twoMonthBefore.BeforeDate, twoMonthBefore.AfterDate)

		monthlyAnalytics(thisMonthData)
		monthlyAnalytics(lastMonthData)
		monthlyAnalytics(twoMonthBeforeData)
	},
}

func sendAPIRequest(beforeDate, afterDate time.Time) Response {
	var monthlyData Response
	for {
		rawURL := fmt.Sprintf("https://api.bitrise.io/v0.1/apps/%s/builds?before=%v&after=%v&limit=10&next=%v", os.Getenv("APP_SLUG_ID"), beforeDate.Unix(), afterDate.Unix(), monthlyData.Paging.Next)
		u, _ := url.Parse(rawURL)

		params := &apiParams{
			method: "GET",
			url:    u,
			header: os.Getenv("ACCESS_TOKEN"),
		}

		ac := newAPIClient()
		_, str, err := ac.doRequest(params)
		if err != nil {
			log.Fatalln(err)
		}

		var data Response
		if err := json.Unmarshal([]byte(str), &data); err != nil {
			log.Fatal(err)
		}

		if len(data.Data) == 0 {
			return Response{}
		}

		for _, hoge := range data.Data {
			monthlyData.Data = append(monthlyData.Data, hoge)
		}
		monthlyData.Paging = data.Paging

		if monthlyData.Paging.Next == "" {
			break
		}
	}

	return monthlyData
}

func monthlyAnalytics(data Response) {
	// dataが１件もない月は出力せず終了
	if len(data.Data) == 0 {
		return
	}

	var buildSumDuration time.Duration
	var buildTimesStatusOK, buildTimesStatusError, buildTimesStatusAborted, buildTotalDays int
	var targetDate time.Time

	for _, buildData := range data.Data {
		if targetDate.Truncate(time.Hour*24) != buildData.StartedOnWorkerAt.Truncate(time.Hour*24) && buildData.Status != 3 {
			buildTotalDays++
		}

		switch buildData.Status {
		case 1:
			buildTimesStatusOK++
			buildSumDuration += buildData.FinishedAt.Sub(buildData.StartedOnWorkerAt)
			targetDate = buildData.StartedOnWorkerAt
		case 2:
			buildTimesStatusError++
			buildSumDuration += buildData.FinishedAt.Sub(buildData.StartedOnWorkerAt)
			targetDate = buildData.StartedOnWorkerAt
		case 3:
			buildTimesStatusAborted++
		}
	}
	avarageTime := int(buildSumDuration) / (buildTimesStatusOK + buildTimesStatusError)
	avarageBuildTimes := data.Paging.TotalItemCount / buildTotalDays
	avarageDailyTime := int(buildSumDuration) / buildTotalDays

	fmt.Printf("--------- analytics:%v月 ---------\n", int(data.Data[0].StartedOnWorkerAt.Month()))
	fmt.Printf("ビルド回数(total)：%v回\n", data.Paging.TotalItemCount)
	fmt.Printf("  > OK：%v回\n", buildTimesStatusOK)
	fmt.Printf("  > Error：%v回\n", buildTimesStatusError)
	fmt.Printf("  > Aborted：%v回\n", buildTimesStatusAborted)
	fmt.Printf("ビルド合計時間：%v\n", buildSumDuration)
	fmt.Printf("1回あたりのビルド平均タイム：%v\n\n", time.Duration(avarageTime))
	fmt.Printf("1日あたりのビルド平均回数：%v\n", avarageBuildTimes)
	fmt.Printf("1日あたりのビルド平均時間：%v\n", time.Duration(avarageDailyTime))
	fmt.Printf("--------- analytics:%v月 ---------\n\n", int(data.Data[0].StartedOnWorkerAt.Month()))
}

func init() {
	rootCmd.AddCommand(monthlyCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// getCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// getCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

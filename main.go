package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/hasura/go-graphql-client"
	"golang.org/x/oauth2/clientcredentials"
)

type QueryGuild struct {
	ReportData struct {
		Reports struct {
			Has_more_pages graphql.Boolean
			Data           []Report
		} `graphql:"reports(guildName: $guild, guildServerSlug: $server, guildServerRegion: $region, page: $page)"`
	}
}

type Report struct {
	Code      graphql.String
	StartTime graphql.Float
	Fights    []Fight
}

type Fight struct {
	Name graphql.String
	//Id         graphql.Int
	Difficulty graphql.Int
	EndTime    graphql.Float
	Kill       graphql.Boolean
}

func main() {
	id := os.Getenv("CLIENT_ID")
	secret := os.Getenv("CLIENT_SECRET")

	useGuild := flag.Bool("guild", false, "Use Guild search")
	useUser := flag.Bool("user", false, "Use user search")

	guildName := flag.String("name", "", "Name of user or guild")
	regionName := flag.String("region", "", "Region")
	serverName := flag.String("server", "", "Server")

	useCsv := flag.Bool("csv", false, "Output to csv")
	fileName := flag.String("fn", "output", "output file name")

	flag.Parse()

	if !(*useGuild || *useUser) {
		fmt.Println("Must specify either user or guild")
		return
	}

	if *useUser {
		panic("user not yet suported")
	}

	conf := clientcredentials.Config{
		ClientID:     id,
		ClientSecret: secret,
		TokenURL:     "https://www.warcraftlogs.com/oauth/token",
	}
	httpClient := conf.Client(context.Background())
	gqlClient := graphql.NewClient("https://www.warcraftlogs.com/api/v2/client", httpClient)

	var reports []Report

	morePages := true
	for page := int8(1); morePages; page++ {
		fmt.Println("Fetch page ", page)
		result, resErr := getGuildPage(gqlClient, *guildName, *serverName, *regionName, page)
		if resErr != nil {
			panic(resErr.Error())
		}
		//fmt.Println(result)
		reports = append(reports, result.ReportData.Reports.Data...)
		morePages = bool(result.ReportData.Reports.Has_more_pages)
	}

	fightStats := NewFightStats()

	for _, report := range reports {
		for _, fight := range report.Fights {
			//fmt.Println(calculateActualTime(int64(report.StartTime), int64(fight.EndTime)))
			_, fightEnd := calculateActualTime(int64(report.StartTime), int64(fight.EndTime))
			fightStats.ProcessFight(fightEnd, fight)
		}
	}

	if len(*fileName) > 0 {
		file, fileErr := os.Create(*fileName)
		if fileErr != nil {
			panic(fileErr.Error())
		}
		defer file.Close()
		outBuf := bufio.NewWriter(file)
		if *useCsv {
			fightStats.CSV(outBuf)
		} else {
			outBuf.WriteString(fightStats.String())
		}
		outBuf.Flush()
	} else {
		fmt.Println(fightStats)
	}
}

func getGuildPage(client *graphql.Client, guildName string, guildServer string, guildRegion string, page int8) (QueryGuild, error) {
	var query QueryGuild
	variables := map[string]any{
		"guild":  graphql.String(guildName),
		"server": graphql.String(guildServer),
		"region": graphql.String(guildRegion),
		"page":   graphql.Int(page),
	}
	if e := client.Query(context.Background(), &query, variables); e != nil {
		return QueryGuild{}, e
	}
	return query, nil
}

func calculateActualTime(startStamp, endDelta int64) (start time.Time, end time.Time) {
	var startTime, endTime time.Time

	startTime = time.UnixMilli(startStamp)
	endTime = startTime.Add(time.Duration(endDelta) * time.Millisecond)
	return startTime, endTime
}

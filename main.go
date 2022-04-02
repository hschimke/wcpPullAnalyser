package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
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

type fightDifficulty int32

func (fd fightDifficulty) String() (difficultyName string) {
	switch fd {
	case 2:
		difficultyName = "raidfinder"
	case 3:
		difficultyName = "normal"
	case 4:
		difficultyName = "heroic"
	case 5:
		difficultyName = "mythic"
	default:
		difficultyName = "unknown"
	}
	return difficultyName
}

type FightKillTime struct {
	Difficulty fightDifficulty
	FirstSeen  time.Time
	FirstKill  time.Time
	Killed     bool
	Count      uint64
}

func (fkt FightKillTime) String() string {
	killText := "--"
	if fkt.Killed {
		killText = fmt.Sprint(fkt.FirstKill)
	}
	return fmt.Sprint(fkt.FirstSeen, " [X] ", killText, " :: ", fkt.Count)
}

type FightStats struct {
	Name string
	data map[string]map[fightDifficulty]FightKillTime
}

func NewFightStats() *FightStats {
	return &FightStats{
		data: make(map[string]map[fightDifficulty]FightKillTime),
	}
}

func (fs *FightStats) ProcessFight(endTime time.Time, fight Fight) {
	diff := fightDifficulty(fight.Difficulty)
	name := string(fight.Name)
	if _, fndName := fs.data[name]; !fndName {
		fs.data[name] = make(map[fightDifficulty]FightKillTime)
	}
	if _, fnd := fs.data[name][diff]; !fnd {
		fs.data[name][diff] = FightKillTime{
			Difficulty: diff,
			Count:      0,
			FirstSeen:  time.Now().Add(time.Hour * 300),
			FirstKill:  time.Now().Add(time.Hour * 300),
			Killed:     false,
		}
	}
	itm := fs.data[name][diff]

	if itm.FirstSeen.After(endTime) {
		itm.FirstSeen = endTime
	}
	if bool(fight.Kill) && itm.FirstKill.After(endTime) {
		itm.FirstKill = endTime
		itm.Killed = true
	}
	itm.Count++

	fs.data[name][diff] = itm
}

func (fs FightStats) String() string {
	var sb strings.Builder

	for name, fdMap := range fs.data {
		sb.WriteString(name)
		sb.WriteString("\n")
		for dif, st := range fdMap {
			sb.WriteString("\t")
			sb.WriteString(dif.String())
			sb.WriteString(": ")
			sb.WriteString(st.String())
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

func (fs FightStats) CSV(w io.Writer) error {
	panic("unimplemented")
}

func main() {
	id := os.Getenv("CLIENT_ID")
	secret := os.Getenv("CLIENT_SECRET")
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
		result, resErr := getPage(gqlClient, page)
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

	fmt.Println(fightStats)
}

func getPage(client *graphql.Client, page int8) (QueryGuild, error) {
	var query QueryGuild
	variables := map[string]any{
		"guild":  graphql.String("hooac"),
		"server": graphql.String("hyjal"),
		"region": graphql.String("us"),
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

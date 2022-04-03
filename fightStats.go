package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"strings"
	"time"

	"golang.org/x/exp/slices"
)

type fightDifficulty int32

func (fd fightDifficulty) String() (difficultyName string) {
	switch fd {
	case 2:
		difficultyName = "raid finder"
	case 3:
		difficultyName = "normal"
	case 4:
		difficultyName = "heroic"
	case 5:
		difficultyName = "mythic"
	default:
		difficultyName = fmt.Sprint("unknown (", int32(fd), ")")
	}
	return difficultyName
}

func (fd fightDifficulty) GetCsvHeaderRows() []string {
	return []string{
		fmt.Sprint(fd, " First Seen"),
		fmt.Sprint(fd, " First Kill"),
		fmt.Sprint(fd, " Encounters"),
		fmt.Sprint(fd, " Kills"),
	}
}

type FightKillTime struct {
	Difficulty fightDifficulty
	FirstSeen  time.Time
	FirstKill  time.Time
	Killed     bool
	Count      uint64
	Kills      uint64
}

func (fkt FightKillTime) String() string {
	killText := "--"
	if fkt.Killed {
		killText = fmt.Sprint(fkt.FirstKill)
	}
	return fmt.Sprint(fkt.FirstSeen, " [X] ", killText, " :: ", fkt.Kills, "/", fkt.Count)
}

type FightStats struct {
	data         map[string]map[fightDifficulty]FightKillTime
	difficulties []fightDifficulty
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
		itm.Kills++
	}
	itm.Count++

	fs.data[name][diff] = itm

	if !slices.Contains(fs.difficulties, diff) {
		fs.difficulties = append(fs.difficulties, diff)
	}
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
	header := []string{
		"Encounter",
	}
	for _, dif := range fs.difficulties {
		header = append(header, dif.GetCsvHeaderRows()...)
	}
	csvWriter := csv.NewWriter(w)
	csvWriter.Write(header[:])
	for name, difMap := range fs.data {
		row := make([]string, 0, 21)
		row = append(row, name)

		for _, dif := range fs.difficulties {
			if fStat, fnd := difMap[dif]; fnd {
				row = append(row, fStat.FirstSeen.String(), fStat.FirstKill.String(), fmt.Sprint(fStat.Count), fmt.Sprint(fStat.Kills))
			} else {
				row = append(row, "", "", "0", "0")
			}
		}

		csvWriter.Write(row)
	}
	csvWriter.Flush()

	return nil
}

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/antihax/goesi"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"
)

func main() {
	email := os.Args[1]
	targetMonth, err := strconv.Atoi(os.Args[2])
	targetYear, err := strconv.Atoi(os.Args[3])

	if err != nil {
		panic(err)
	}

	var zkills []ZkillMailEntry
	page := 1
	for {
		var zkillsPage []ZkillMailEntry
		fmt.Println(fmt.Sprintf("https://zkillboard.com/api/corporationID/98732555/year/%d/month/%d/page/%d/", targetYear, targetMonth, page))
		resp, err := http.Get(fmt.Sprintf("https://zkillboard.com/api/corporationID/98732555/year/%d/month/%d/page/%d/", targetYear, targetMonth, page))
		if err != nil {
			panic(err)
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			panic(err)
		}
		json.Unmarshal(body, &zkillsPage)
		if len(zkillsPage) == 0 {
			break
		}
		zkills = append(zkills, zkillsPage...)
		page++
	}
	var killmailsOutputs [][]KillMailOutput

	// create ESI client
	var ctx context.Context
	client := http.Client{}
	esiClient := goesi.NewAPIClient(&client, email)

	fmt.Println("Beginning processing of killmails")
	for _, value := range zkills {
		fmt.Println(value)
		//killmail, _, err := esiClient.ESI.KillmailsApi.GetKillmailsKillmailIdKillmailHash(ctx, value.KillmailHash, value.KillmailId, nil)
		killmail, _, err := esiClient.ESI.KillmailsApi.GetKillmailsKillmailIdKillmailHash(ctx, value.ZKB.KillmailHash, value.KillmailId, nil)
		if err != nil {
			panic(err)
		}

		if int(killmail.KillmailTime.Month()) != targetMonth {
			continue
		}

		victim, _, err := esiClient.ESI.CharacterApi.GetCharactersCharacterId(ctx, killmail.Victim.CharacterId, nil)
		if victim.CorporationId == 98732555 {
			continue
		}

		victimShip, _, err := esiClient.ESI.UniverseApi.GetUniverseTypesTypeId(ctx, killmail.Victim.ShipTypeId, nil)

		solarSystem, _, err := esiClient.ESI.UniverseApi.GetUniverseSystemsSystemId(ctx, killmail.SolarSystemId, nil)

		killmailOutputs := make([]KillMailOutput, len(killmail.Attackers))
		for index, charValue := range killmail.Attackers {
			if charValue.CharacterId == 0 {
				killmailOutputs[index] = KillMailOutput{
					VictimId:            killmail.Victim.CharacterId,
					VictimName:          victim.Name,
					VictimShipId:        killmail.Victim.ShipTypeId,
					VictimShipName:      victimShip.Name,
					AttackerId:          charValue.CharacterId,
					AttackerName:        "npc",
					AttackerShipId:      charValue.ShipTypeId,
					AttackerShipName:    "npc",
					AttackerCorporateId: 0,
					AttackerAllianceId:  0,
					TotalDamage:         charValue.DamageDone,
					FinalBlow:           charValue.FinalBlow,
					NumAttackers:        len(killmail.Attackers),
					SolarSystemId:       killmail.SolarSystemId,
					SolarSystemName:     solarSystem.Name,
					SecurityLevel:       solarSystem.SecurityStatus,
					KillMailTime:        killmail.KillmailTime,
					TotalValue:          value.ZKB.TotalValue,
				}
				continue
			}
			character, _, err := esiClient.ESI.CharacterApi.GetCharactersCharacterId(ctx, charValue.CharacterId, nil)
			if err != nil && err.Error() != "404 Not Found" {
				panic(err)
			} else if err != nil && err.Error() == "404 Not Found" {
				continue
			}
			if character.AllianceId != 99002217 {
				continue
			}
			attackerShip, _, err := esiClient.ESI.UniverseApi.GetUniverseTypesTypeId(ctx, charValue.ShipTypeId, nil)
			killmailOutputs[index] = KillMailOutput{
				VictimId:            killmail.Victim.CharacterId,
				VictimName:          victim.Name,
				VictimShipId:        killmail.Victim.ShipTypeId,
				VictimShipName:      victimShip.Name,
				AttackerId:          charValue.CharacterId,
				AttackerName:        character.Name,
				AttackerShipId:      charValue.ShipTypeId,
				AttackerShipName:    attackerShip.Name,
				AttackerCorporateId: character.CorporationId,
				AttackerAllianceId:  character.AllianceId,
				TotalDamage:         charValue.DamageDone,
				FinalBlow:           charValue.FinalBlow,
				NumAttackers:        len(killmail.Attackers),
				SolarSystemId:       killmail.SolarSystemId,
				SolarSystemName:     solarSystem.Name,
				SecurityLevel:       solarSystem.SecurityStatus,
				KillMailTime:        killmail.KillmailTime,
				TotalValue:          value.ZKB.TotalValue,
			}
		}
		killmailsOutputs = append(killmailsOutputs, killmailOutputs)
	}

	fmt.Println("Writing data to file")
	WriteToFile(killmailsOutputs)
}

func WriteToFile(killmailsOutputs [][]KillMailOutput) {
	f, err := os.Create("killmails.csv")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	if _, err := f.WriteString("victimId,victimName,victimShipId,victimShipName,attackerId,attackerName,attackerShipId,attackerShipName,totalDamage,finalBlow,numAttackers,solarSystemId,solarSystemName,securityLevel,killmailTime,totalValue\n"); err != nil {
		panic(err)
	}
	f.Sync()

	for _, output := range killmailsOutputs {
		for _, killmail := range output {
			if killmail.AttackerId == 0 && killmail.AttackerName != "npc" {
				continue
			}
			line := fmt.Sprintf("%d,%s,%d,%s,%d,%s,%d,%s,%d,%d,%d,%t,%d,%d,%s,%f,%v,%f", killmail.VictimId, killmail.VictimName, killmail.VictimShipId, killmail.VictimShipName, killmail.AttackerId, killmail.AttackerName, killmail.AttackerShipId, killmail.AttackerShipName, killmail.AttackerCorporateId, killmail.AttackerAllianceId, killmail.TotalDamage, killmail.FinalBlow, killmail.NumAttackers, killmail.SolarSystemId, killmail.SolarSystemName, killmail.SecurityLevel, killmail.KillMailTime, killmail.TotalValue)
			if _, err := f.WriteString(line + "\n"); err != nil {
				panic(err)
			}
			f.Sync()
		}
	}
}

type KillMailEntry struct {
	KillmailHash string `json:"killmail_hash"`
	KillmailId   int32  `json:"killmail_id"`
}

// if _, err := f.WriteString("victimId,victimName,victimShipId,victimShipName,attackerId,attackerName,attackerShipId,attackerShipName,totalDamage,finalBlow,numAttackers,solarSystemId,solarSystemName,killmailTime\n"); err != nil {
type KillMailOutput struct {
	VictimId            int32
	VictimName          string
	VictimShipId        int32
	VictimShipName      string
	AttackerId          int32
	AttackerName        string
	AttackerShipId      int32
	AttackerShipName    string
	AttackerCorporateId int32
	AttackerAllianceId  int32
	TotalDamage         int32
	FinalBlow           bool
	NumAttackers        int
	SolarSystemId       int32
	SolarSystemName     string
	SecurityLevel       float32
	KillMailTime        time.Time
	TotalValue          float64
}

type ZKB struct {
	KillmailHash string  `json:"hash"`
	TotalValue   float64 `json:"totalValue"`
}

type ZkillMailEntry struct {
	KillmailId int32 `json:"killmail_id"`
	ZKB        ZKB   `json:"zkb"`
}

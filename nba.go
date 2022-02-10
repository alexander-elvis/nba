package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sync"
)

var (
	api       = "https://www.balldontlie.io/api/v1/"
	playerApi = api + "players?search=%s"
	statsApi  = api + "stats?player_ids[]=%d"
)

// go build nba.go; ./nba player -name=lebron
// go build nba.go; ./nba buckets

func main() {
	if len(os.Args) < 2 {
		log.Fatalln("Please provide command")
	}
	switch os.Args[1] {
	case "player":
		handlePlayerCMD()
	case "buckets":
		handleBucketsCMD()
	default:
		fmt.Println("Command not recognized:", os.Args[1])
	}

}

func handleBucketsCMD() {
	var playerIds = []int{237, 237, 237, 237, 237, 237, 237, 237, 237, 237, 237, 237, 237, 237, 237, 237, 237, 237, 237, 237, 237, 237, 237, 237, 237, 237, 237, 237, 237, 237}
	var concurrencyCount = len(playerIds)

	var playerIdChannel = make(chan int)
	var pointsChannel = make(chan []int)
	var averagePointsChannel = make(chan int)

	var playerIdChannelWG sync.WaitGroup
	var pointsChannelWG sync.WaitGroup
	var averagePointsChannelWG sync.WaitGroup

	playerIdChannelWG.Add(len(playerIds))
	pointsChannelWG.Add(len(playerIds))
	averagePointsChannelWG.Add(len(playerIds))

	go func() {
		playerIdChannelWG.Wait()
		close(playerIdChannel)
		pointsChannelWG.Wait()
		close(pointsChannel)
		averagePointsChannelWG.Wait()
		close(averagePointsChannel)
	}()

	go func() {
		for _, id := range playerIds {
			playerIdChannel <- id
		}
	}()

	for i := 1; i <= concurrencyCount; i++ {
		scheduleStatsRetrieval(&playerIdChannel, &pointsChannel, &playerIdChannelWG)
		scheduleStatsCalculation(&pointsChannel, &averagePointsChannel, &pointsChannelWG)
	}

	var average = 0
	var averageCount = 0
	for averagePoint := range averagePointsChannel {
		averageCount++
		average += averagePoint
		averagePointsChannelWG.Done()
	}

	if averageCount == 0 {
		fmt.Println("No records found!")
		return
	}
	fmt.Println("Average PPG: ", average/averageCount)
}

func scheduleStatsCalculation(pointsChannel *chan []int, averagePointsChannel *chan int, wg *sync.WaitGroup) {
	go func() {
		for points := range *pointsChannel {
			var sum = 0
			for _, point := range points {
				sum += point
			}
			*averagePointsChannel <- sum / len(points)
			wg.Done()
		}
	}()
}

func scheduleStatsRetrieval(playerIdChannel *chan int, pointsChannel *chan []int, wg *sync.WaitGroup) {
	go func() {
		for playerId := range *playerIdChannel {
			var response = new(StatsResponse)
			var err = getRequest(fmt.Sprintf(statsApi, playerId), response)

			if err != nil {
				log.Printf("Failed to retrieve stats for %d err: %v\n", playerId, err)
				continue
			}
			var points = make([]int, len(response.Data))
			for point := range response.Data {
				points = append(points, point)
			}
			if len(points) > 1 {
				*pointsChannel <- points
			} else {
				fmt.Println("ignoring response", playerId)
			}
			wg.Done()
		}
	}()
}

func handlePlayerCMD() {
	var playerCmd = flag.NewFlagSet("player", flag.ExitOnError)
	var name = playerCmd.String("name", "", "search string")
	var err = playerCmd.Parse(os.Args[2:])
	if err != nil {
		log.Fatalf("Failed to parse command %s", err)
	}
	var p = new(PlayerResponse)
	err = getRequest(fmt.Sprintf(playerApi, *name), p)
	if err != nil {
		log.Fatalf("Failed to retrieve player info %s", err)
	}
	for _, player := range p.Data {
		fmt.Printf("%-5d\t%-10s\t%-10s\t%-10s\n", player.Id, player.FirstName, player.LastName, player.Team.FullName)
	}
}

type StatsResponse struct {
	Data []struct {
		Pts int `json:"pts"`
	}
}
type PlayerResponse struct {
	Data []struct {
		Id        int    `json:"id"`
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		Team      struct {
			FullName string `json:"full_name"`
		}
	}
}

func getRequest(url string, r interface{}) error {
	var err error

	resp, err := http.Get(url)
	if err != nil {
		return err
	}

	defer func() {
		err = resp.Body.Close()
		if err != nil {
			log.Println("Failed to close response body")
		}
	}()

	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	err = json.Unmarshal(bytes, r)

	if err != nil {
		return fmt.Errorf("failed to deserialize %v", err)
	}

	return nil
}

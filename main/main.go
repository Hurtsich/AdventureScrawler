package main

import (
	"bytes"
	"encoding/base64"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"github.com/go-resty/resty/v2"
	"math/rand"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"
)

var (
	Client       = resty.New()
	Shop         []string
	ItemsList    []Item
	DungeonsList []string
	Logs         []DungeonLog

	BestLogs map[string]Log
)

const (
	urlPreparation = "https://adventure-crawler.up.railway.app/preparation"
	urlInscription = "https://adventure-crawler.up.railway.app/inscription"
	urlScoreBoard  = "https://adventure-crawler.up.railway.app/score-board"
	urlDungeons    = "https://adventure-crawler.up.railway.app/preparation/adventures"
	urlItems       = "https://adventure-crawler.up.railway.app/preparation/items"
	urlBackpack    = "https://adventure-crawler.up.railway.app/preparation/backpack"
	urlExploration = "https://adventure-crawler.up.railway.app/exploration/adventures"
)

func main() {
	headersGet := map[string]string{
		"Accept": "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8",
	}
	resp, err := Get(urlScoreBoard, headersGet)
	if err != nil {
		fmt.Printf("get url %s: %v", urlScoreBoard, err)
	}
	if !strings.Contains(string(resp.Body()), "Justin") {
		_, err = Post(urlInscription, headersGet, Authentication{"Justin", "justin"})
		if err != nil {
			fmt.Printf("post url %s: %v", urlInscription, err)
		}
	}
	_, err = Get(urlPreparation, headersGet)
	if err != nil {
		fmt.Printf("get url %s: %v", urlPreparation, err)
	}
	headersGet["Authorization"] = GetAuth()
	err = GetItems(headersGet)
	if err != nil {
		fmt.Printf("get url %s: %v", urlItems, err)
	}
	err = GetDungeons(headersGet)
	if err != nil {
		fmt.Printf("get url %s: %v", urlDungeons, err)
	}

	fmt.Printf("Nombre d'aventures: %d\n", len(DungeonsList))
	fmt.Printf("Nombre d'items: %d", len(Shop))
	exportItems(ItemsList)
	//Crawl(headersGet)
	//BestExploration(headersGet)
	//BasicExploration(headersGet)
	//Exploration(headersGet)
	Bulk(headersGet)
}

func Bulk(headers map[string]string) {
	var items []string
	for i := 0; i < 10; i++ {
		items = append(items, Shop[rand.Intn(len(Shop))])
	}
	_, err := SupplyBackpack(NewBackpack(items...), headers)
	if err != nil {
		fmt.Printf("supply backpack: %v", err)
	}
	for _, adventure := range DungeonsList {
		resp, err := ExploreDungeons(adventure, headers)
		if err != nil {
			fmt.Printf("explore %s: %v", adventure, err)
		}
		ExploitResponse(resp, adventure, items)
	}
	exportLogs(Logs, "../data/bulk.csv")
}

func Exploration(headers map[string]string) {
	items := []string{"Champignon", "Arc", "Armure en pierre", "Grappin", "Torche", "Kit de premier soins", "Bouclier en fer"}
	_, err := SupplyBackpack(NewBackpack(items...), headers)
	if err != nil {
		fmt.Printf("supply backpack: %v", err)
	}
	resp, err := ExploreDungeons("addo-solitudo-voluptatem-demo-theatrum-apostolus", headers)
	if err != nil {
		fmt.Printf("explore addo-solitudo-voluptatem-demo-theatrum-apostolus: %v", err)
	}
	ExploitResponse(resp, "addo-solitudo-voluptatem-demo-theatrum-apostolus", items)
	resp, err = ExploreDungeons("553", headers)
	if err != nil {
		fmt.Printf("explore 553: %v", err)
	}
	ExploitResponse(resp, "553", items)
	resp, err = ExploreDungeons("1622", headers)
	if err != nil {
		fmt.Printf("explore 1622: %v", err)
	}
	ExploitResponse(resp, "1622", items)
	resp, err = ExploreDungeons("undique-ancilla-uberrime-sol-quas-contego", headers)
	if err != nil {
		fmt.Printf("explore undique-ancilla-uberrime-sol-quas-contego: %v", err)
	}
	ExploitResponse(resp, "undique-ancilla-uberrime-sol-quas-contego", items)
	resp, err = ExploreDungeons("addo-solitudo-voluptatem-demo-theatrum-apostolus", headers)
	if err != nil {
		fmt.Printf("explore addo-solitudo-voluptatem-demo-theatrum-apostolus: %v", err)
	}
	ExploitResponse(resp, "addo-solitudo-voluptatem-demo-theatrum-apostolus", items)
	resp, err = ExploreDungeons("calamitas-tumultus-articulus-titulus-coerceo-vulgo", headers)
	if err != nil {
		fmt.Printf("explore calamitas-tumultus-articulus-titulus-coerceo-vulgo: %v", err)
	}
	ExploitResponse(resp, "calamitas-tumultus-articulus-titulus-coerceo-vulgo", items)
	resp, err = ExploreDungeons("addo-solitudo-voluptatem-demo-theatrum-apostolus", headers)
	if err != nil {
		fmt.Printf("explore addo-solitudo-voluptatem-demo-theatrum-apostolus: %v", err)
	}
	ExploitResponse(resp, "addo-solitudo-voluptatem-demo-theatrum-apostolus", items)
	exportLogs(Logs, "../data/addo-solitudo-voluptatem-demo-theatrum-apostolus.csv")
}

func Crawl(headers map[string]string) {
	bestLogs := getBestLogs()
	BestLogs = make(map[string]Log)
	for _, adventure := range bestLogs {
		bestScore, err := strconv.Atoi(adventure[2])
		if err != nil {
			fmt.Printf("convert score to int %s: %v", adventure[2], err)
		}
		BestLogs[adventure[0]] = Log{
			ItemName: strings.Split(adventure[1], "/"),
			Score:    bestScore,
			Report:   adventure[3],
		}
	}
	tick := time.Tick(2 * time.Minute)
	for _, adventure := range DungeonsList {
		var log Log
		log.ItemName = BestLogs[adventure].ItemName
		log.Score = BestLogs[adventure].Score
		cptRq := ""
		for i := 0; i < 1000; i++ {
			if cptRq == "0" {
				<-tick
			}
			newItems := defineNewItemsList(log.ItemName)
			resp, err := SupplyBackpack(NewBackpack(newItems...), headers)
			if err != nil {
				fmt.Printf("supply backpack with %v: %v", log.ItemName, err)
			}
			cptRq = resp.Header().Get("X-Ratelimit-Remaining")
			resp, err = ExploreDungeons(adventure, headers)
			if err != nil {
				fmt.Printf("explore dungeon %s: %v", adventure, err)
			}
			cptRq = resp.Header().Get("X-Ratelimit-Remaining")
			var currentLog Log
			body := strings.ReplaceAll(string(resp.Body()), `\n`, "")
			err = json.Unmarshal([]byte(body), &currentLog)
			if err != nil {
				fmt.Printf("unmarshal log: %v", err)
			}
			if currentLog.Score > log.Score {
				log = currentLog
			}
		}
		Logs = append(Logs, DungeonLog{adventure, log})
	}
	exportLogs(Logs, "../data/crawl.csv")
}

func defineNewItemsList(items []string) []string {
	var newItems []string
	for _, item := range Shop {
		if !slices.Contains(items, item) {
			if rand.Intn(100) < 15 {
				newItems = append(newItems, item)
			}
		} else {
			if rand.Intn(100) < 50 {
				newItems = append(newItems, item)
			}
		}
	}
	return newItems
}

func BestExploration(headers map[string]string) {
	bestLogs := getBestLogs()
	for _, adventure := range bestLogs {
		items := strings.Split(adventure[1], "/")
		_, err := SupplyBackpack(NewBackpack(items...), headers)
		if err != nil {
			fmt.Printf("supply backpack with %v: %v", items, err)
		}
		adventureName := adventure[0]
		resp, err := ExploreDungeons(adventureName, headers)
		if err != nil {
			fmt.Printf("explore dungeon %s: %v", adventureName, err)
		}
		body := strings.ReplaceAll(string(resp.Body()), `\n`, "")
		var log Log
		err = json.Unmarshal([]byte(body), &log)
		if err != nil {
			fmt.Printf("unmarshal log: %v", err)
		}
		log.ItemName = items
		Logs = append(Logs, DungeonLog{adventureName, log})
	}
	exportLogs(Logs, "../data/best.csv")
}

func exportItems(items []Item) {
	var prettyJSON bytes.Buffer
	logJSON, err := json.Marshal(items)
	if err != nil {
		fmt.Printf("marshal Logs: %v", err)
	}
	if err = json.Indent(&prettyJSON, logJSON, "", "    "); err != nil {
		fmt.Printf("pretty Logs: %v", err)
	}
	fmt.Println(prettyJSON.String())

	logFile, err := os.Create("../data/items.csv")
	if err != nil {
		fmt.Printf("create log logFile: %v", err)
	}
	defer logFile.Close()

	writer := csv.NewWriter(logFile)
	defer writer.Flush()

	for _, item := range items {
		err = writer.Write([]string{item.Name, item.Description})
		if err != nil {
			fmt.Printf("writing log line: %v", err)
		}
	}
}

func exportLogs(logs []DungeonLog, filename string) {
	var prettyJSON bytes.Buffer
	logJSON, err := json.Marshal(logs)
	if err != nil {
		fmt.Printf("marshal Logs: %v", err)
	}
	if err = json.Indent(&prettyJSON, logJSON, "", "    "); err != nil {
		fmt.Printf("pretty Logs: %v", err)
	}
	fmt.Println(prettyJSON.String())

	logFile, err := os.Create(filename)
	if err != nil {
		fmt.Printf("create log logFile: %v", err)
	}
	defer logFile.Close()

	writer := csv.NewWriter(logFile)
	defer writer.Flush()

	for _, log := range logs {
		err = writer.Write([]string{log.DungeonName, strings.Join(log.Summary.ItemName, "/"), fmt.Sprint(log.Summary.Score), log.Summary.Report})
		if err != nil {
			fmt.Printf("writing log line: %v", err)
		}
	}
}

func getBestLogs() [][]string {
	file, err := os.Open("../data/best.csv")
	if err != nil {
		fmt.Printf("open best Logs file: %v", err)
		return nil
	}
	defer file.Close()
	reader := csv.NewReader(file)
	bestLogs, err := reader.ReadAll()
	fmt.Println(bestLogs)
	if err != nil {
		fmt.Printf("read best Logs: %v", err)
		return nil
	}
	return bestLogs
}

func BasicExploration(headers map[string]string) {
	err := Explore(headers)
	if err != nil {
		fmt.Printf("EXPLORE: %v", err)
	}

	exportLogs(Logs, "../data/logs.csv")
}

func Explore(headers map[string]string) error {
	tick := time.Tick(2 * time.Minute)
	cptRq := ""
	for _, dungeon := range DungeonsList {
		//for _, item := range Shop {
		item := Shop[rand.Intn(len(Shop))]
		if cptRq == "0" {
			<-tick
		}
		resp, err := SupplyBackpack(NewBackpack(item), headers)
		if err != nil {
			fmt.Printf("supply backpack with %s: %v", item, err)
		}
		cptRq = resp.Header().Get("X-Ratelimit-Remaining")
		resp, err = ExploreDungeons(dungeon, headers)
		if err != nil {
			fmt.Printf("explore dungeon %s: %v", dungeon, err)
		}
		cptRq = resp.Header().Get("X-Ratelimit-Remaining")
		ExploitResponse(resp, dungeon, []string{item})
		//}
	}
	return nil
}

func ExploitResponse(resp *resty.Response, dungeon string, items []string) {
	body := strings.ReplaceAll(string(resp.Body()), `\n`, "")
	var log Log
	err := json.Unmarshal([]byte(body), &log)
	if err != nil {
		fmt.Printf("unmarshal log: %v", err)
	}
	log.ItemName = append(log.ItemName, items...)
	Logs = append(Logs, DungeonLog{dungeon, log})
}

func GetItems(headers map[string]string) error {
	totalItems := -1
	var s Items
	url := ""
	for len(Shop) != totalItems {
		if s.Next != "" {
			url = urlItems + s.Next
		} else {
			url = urlItems
		}
		resp, err := Get(url, headers)
		if err != nil {
			fmt.Printf("get url %s: %v", url, err)
		}
		err = json.Unmarshal(resp.Body(), &s)

		for _, item := range s.Items {
			Shop = append(Shop, item.Name)
			ItemsList = append(ItemsList, item)
		}
		totalItems = s.Total
	}
	return nil
}

func GetDungeons(headers map[string]string) error {
	totalAdventures := -1
	var l Adventures
	url := ""
	for len(DungeonsList) != totalAdventures {
		if l.Next != "" {
			url = urlDungeons + l.Next
		} else {
			url = urlDungeons
		}

		resp, err := Get(url, headers)
		if err != nil {
			fmt.Printf("get url %s: %v", url, err)
		}
		err = json.Unmarshal(resp.Body(), &l)

		for _, dungeon := range l.Adventures {
			DungeonsList = append(DungeonsList, dungeon.Name)
		}
		totalAdventures = l.Total
	}
	return nil
}

func ExploreDungeons(dungeonName string, headers map[string]string) (*resty.Response, error) {
	return Post(fmt.Sprintf("%s/%s", urlExploration, dungeonName), headers, nil)
}

func SupplyBackpack(items Backpack, headers map[string]string) (*resty.Response, error) {
	resp, err := Post(urlBackpack, headers, items)
	return resp, err
}

func Get(url string, headers map[string]string) (*resty.Response, error) {
	fmt.Println("GET ", url)
	resp, err := Client.R().EnableTrace().
		SetHeaders(headers).
		Get(url)
	if err != nil {
		return nil, err
	}
	err = PrintResponse(err, resp)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func Post(url string, headers map[string]string, arg interface{}) (*resty.Response, error) {
	headers["Content-Type"] = "application/json"

	fmt.Println("POST ", url)
	resp, err := Client.R().EnableTrace().
		SetHeaders(headers).
		SetBody(arg).
		Post(url)
	if err != nil {
		return nil, err
	}
	err = PrintResponse(err, resp)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func PrintResponse(err error, resp *resty.Response) error {
	body := fmt.Sprint(resp)
	if IsJSON(body) {
		var prettyJSON bytes.Buffer
		if err = json.Indent(&prettyJSON, []byte(body), "", "    "); err != nil {
			return err
		}
		body = prettyJSON.String()
	}

	// Explore response object
	fmt.Println("Response Info:")
	fmt.Println("  Error         :", err)
	fmt.Println("  Status Code   :", resp.StatusCode())
	fmt.Println("  Status        :", resp.Status())
	fmt.Println("  Proto         :", resp.Proto())
	fmt.Println("  Time          :", resp.Time())
	fmt.Println("  Headers       :", resp.Header())
	fmt.Println("  Received At   :", resp.ReceivedAt())
	fmt.Println("  Body          :\n", body)
	fmt.Println()

	//// Explore trace info
	//fmt.Println("Request Trace Info:")
	//ti := resp.Request.TraceInfo()
	//fmt.Println("  DNSLookup     :", ti.DNSLookup)
	//fmt.Println("  ConnTime      :", ti.ConnTime)
	//fmt.Println("  TCPConnTime   :", ti.TCPConnTime)
	//fmt.Println("  TLSHandshake  :", ti.TLSHandshake)
	//fmt.Println("  ServerTime    :", ti.ServerTime)
	//fmt.Println("  ResponseTime  :", ti.ResponseTime)
	//fmt.Println("  TotalTime     :", ti.TotalTime)
	//fmt.Println("  IsConnReused  :", ti.IsConnReused)
	//fmt.Println("  IsConnWasIdle :", ti.IsConnWasIdle)
	//fmt.Println("  ConnIdleTime  :", ti.ConnIdleTime)
	//fmt.Println("  RequestAttempt:", ti.RequestAttempt)
	//fmt.Println("  RemoteAddr    :", ti.RemoteAddr.String())
	fmt.Println()

	return nil
}

type Authentication struct {
	Name     string `json:"name"`
	Password string `json:"password"`
}

type Backpack struct {
	Items []string `json:"itemsName"`
}

type Items struct {
	Items    []Item `json:"items"`
	Next     string `json:"next"`
	Previous string `json:"previous"`
	Total    int    `json:"total"`
}

type Item struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type Adventures struct {
	Adventures []Dungeon `json:"adventures"`
	Next       string    `json:"next"`
	Previous   string    `json:"previous"`
	Total      int       `json:"total"`
}

type Dungeon struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type Log struct {
	ItemName []string `json:"items"`
	Score    int      `json:"score"`
	Report   string   `json:"report"`
}

type DungeonLog struct {
	DungeonName string `json:"dungeon"`
	Summary     Log    `json:"summary"`
}

func NewBackpack(items ...string) Backpack {
	return Backpack{items}
}

func GetAuth() string {
	auth := Authentication{"Justin", "justin"}
	data := fmt.Sprintf("%s:%s", auth.Name, auth.Password)
	b64String := base64.StdEncoding.EncodeToString([]byte(data))
	return fmt.Sprintf("%s %s", "Basic", b64String)
}

func IsJSON(str string) bool {
	var js json.RawMessage
	return json.Unmarshal([]byte(str), &js) == nil
}

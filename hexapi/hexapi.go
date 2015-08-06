package main

//  Things needed to do
//  + Parse JSON into something I can do stuff with
//  + Read in config values from a config file
//  - Get price data from URL
//  - Handle the following types of events
//    - Login
//    - Logout
//    - DraftPack
//    - DraftCardPicked/DaraftCardPicked
//    - GameStarted
//    - GameEnded
//    - PlayerUpdated
//    - CardUpdated
//  - Track pack data and indicate which cards were picked when packs wheel
//  - Track Profit/Loss for drafts
//
//  Stretch Goals
//  - Post card data to remote URL (for collating draw data)
//  - Check for updated versions by checking remote URL

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// The Cards we work with and all the info we need about them
type Card struct {
	name   string
	uuid   string
	qty    int
	rarity string
	gold   int
	plat   int
}

// And this is all our cards together... we use uuid as a key
var cardCollection map[string]Card = make(map[string]Card)

type nameToUuidMap struct {
	name string
	uuid string
}

type sortedCardCollection struct {
	cc map[string]Card
	s  []string
}

var scc sortedCardCollection

// Configuration values we'll use all around
//type Config map[string]string
var Config = make(map[string]string)

// And some general variables we'll use to keep track of things
var GameStartTime time.Time = time.Now()

// FUNCTIONS

func printCollection() {
	// Make a sorted card collection
	scc := sortedCardCollection{cardCollection, make([]string, len(cardCollection))}
	// Populate the sorted card collection
	n := 0
	for _, c := range scc.cc {
		scc.s[n] = c.name
		n += 1
	}
	//		name := c.name
	//		uuid = c.uuid
	//	}
	//	sort.Strings(m)
	//	for _, u := range c {
	sort.Sort(scc)
	for _, entry := range scc.cc {
		if entry.qty > 0 {
			printCardInfo(entry)
		}
	}
}

/// BEGIN SORT STUFF
func (scc sortedCardCollection) Len() int {
	return len(scc.cc)
}

func (scc sortedCardCollection) Less(i, j int) bool {
	a := scc.cc[scc.s[i]]
	b := scc.cc[scc.s[j]]
	return a.name > b.name
}

func (scc sortedCardCollection) Swap(i, j int) {
	scc.s[i], scc.s[j] = scc.s[j], scc.s[i]
}

func sortedKeys(cc map[string]Card) []string {
	scc := new(sortedCardCollection)
	scc.cc = cc
	scc.s = make([]string, len(cc))
	i := 0
	for key, _ := range cc {
		scc.s[i] = key
		i++
	}
	sort.Sort(scc)
	return scc.s
}

/// END SORT STUFF

func cardUpdatedEvent() {
}

func incrementCardCount(uuid string) {
	changeCardCount(uuid, 1)
}
func decrementCardCount(uuid string) {
	changeCardCount(uuid, -1)
}
func changeCardCount(uuid string, i int) {
	if _, ok := cardCollection[uuid]; ok {
		c := cardCollection[uuid]
		c.qty += i
		cardCollection[uuid] = c
	}
}

func printCardInfo(c Card) {
	fmt.Printf("'%v' [Qty: %v] - %vp and %vg\n", c.name, c.qty, c.plat, c.gold)
}

func collectionEvent(f map[string]interface{}) {
	action := f["Action"]
	added, _ := f["CardsAdded"].([]interface{})
	removed, _ := f["CardsRemoved"].([]interface{})
	switch action {
	case "Overwrite":
		fmt.Printf("Got an Overwrite Collection message\n")
		// First thing we do is reset counts on all cards
		for k, v := range cardCollection {
			v.qty = 0
			cardCollection[k] = v
		}
		// Ok, let's extract the cards and update the numbers of each card, then cache that locally (in case we
		// need to restart for some reason)
		for _, u := range added {
			card := u.(map[string]interface{})
			uuid_map := card["Guid"].(map[string]interface{})
			uuid := uuid_map["m_Guid"].(string)
			incrementCardCount(uuid)
		}
		// For DEBUGGING
		printCollection()
	case "Update":
		fmt.Printf("Got an Update Collection message\nCards removed: %v\n", removed)
	default:
		fmt.Printf("Got an unknown Collection Action '%v'\n", action)
	}
}

func draftCardPickedEvent() {
}

func draftPackEvent() {
}

// Message: {"Winners":["Uzume, Grand Concubunny"],"Losers":["Warmaster Fuzzuko"],"User":"InGameName","Message":"GameEnded"}
func gameEndedEvent() {
	elapsed := time.Since(GameStartTime)
	fmt.Printf("Elapsed Game Time: %s", elapsed)
}

// Message: {"Players":[],"User":"InGameName","Message":"GameStarted"}
func gameStartedEvent() {
	GameStartTime = time.Now()
}

// {"User":"","Message":"Login"} - Logging in from new start of Hex
// {"User":"InGameName","Message":"Login"} - Logging over to different account
func loginEvent(s string) {
	if s != "" {
		fmt.Printf("Welcome Back to the world of Entrath, %v! We hope you enjoy your stay.\n", s)
	} else {
		fmt.Printf("Welcome Back to the world of Entrath! We hope you enjoy your stay.\n")
	}
}

func logoutEvent(s string) {
	if s != "" {
		fmt.Printf("Thank you for visiting the world of Entrath, %v. We look forward to seeing you again.\n", s)
	} else {
		fmt.Printf("Thank you for visiting the world of Entrath. We look forward to seeing you again.\n")
	}
}

func playerUpdatedEvent() {
}

func incoming(rw http.ResponseWriter, req *http.Request) {
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		panic("AIEEE: Could not readAll for req.Body")
	}
	fmt.Println("REPLY BODY: ", string(body))
	//err = json.Unmarshal(body, &t)
	var f map[string]interface{}
	err = json.Unmarshal(body, &f)
	if err != nil {
		panic("AIEEE: Could not Unmarshall the body")
	}
	//	fmt.Println("DEBUG: Unmarshall successful")
	fmt.Println("DEBUG: Message is", f["Message"])
	msg := f["Message"]
	switch msg {
	case "CardUpdated":
		fmt.Printf("Got a Card Updated message\n")
		cardUpdatedEvent()
	case "Collection":
		fmt.Printf("Got a Collection message\n")
		collectionEvent(f)
	case "DraftCardPicked":
		fmt.Printf("Got a Draft Card Picked message\n")
		draftCardPickedEvent()
	case "DraftPack":
		fmt.Printf("Got a Draft Pack message\n")
		draftPackEvent()
	case "GameEnded":
		fmt.Printf("Got a Game Ended message\n")
		gameEndedEvent()
	case "GameStarted":
		fmt.Printf("Got a Game Started message\n")
		gameStartedEvent()
	case "Login":
		//		fmt.Printf("Got a Login message\n")
		if user, ok := f["User"].(string); ok {
			loginEvent(user)
		} else {
			loginEvent("")
		}
	case "Logout":
		//		fmt.Printf("Got a Logout message\n")
		if user, ok := f["User"].(string); ok {
			logoutEvent(user)
		} else {
			logoutEvent("")
		}
	case "PlayerUpdated":
		fmt.Printf("Got a Player Updated message\n")
		playerUpdatedEvent()
	default:
		fmt.Printf("Don't know how to handle message '%v'\n", msg)
		//}
		//	for k, v := range f {
		//		fmt.Printf("working on key %v with value %v\n", k, v)
		//		switch vv := v.(type) {
		//		case string:
		//			fmt.Println(k, "is string", vv)
		//		case float64:
		//			fmt.Println(k, "is float64", vv)
		//		case int:
		//			fmt.Println(k, "is int", vv)
		//		case map[string]interface{}:
		//			fmt.Println(k, "is a map", vv)
		//			for i, u := range vv {
		//				fmt.Println("\t", i, u)
		//			}
		//		case []interface{}:
		//			fmt.Println(k, "is an array:")
		//			for i, u := range vv {
		//				fmt.Println("\t", i, u)
		//			}
		//		default:
		//			fmt.Println(k, "is of a type,", reflect.TypeOf(vv), ", I don't know how to handle: ", vv)
		//		}
	}
}

func loadDefaults() map[string]string {
	retMap := make(map[string]string)
	retMap["price_url"] = "http://doc-x.net/hex/all_prices_with_uuids.txt"
	// May be able to get rid of this since we're getting uuids from above
	retMap["aa_promo_url"] = "http://doc-x.net/hex/aa_promo_list.txt"
	retMap["collection_file"] = "collection.out"
	// Here so we can copy and paste it later
	retMap["key"] = "val"

	return retMap
}

func readConfig(fname string, config map[string]string) map[string]string {

	if _, err := os.Stat(fname); err != nil {
		fmt.Printf("No config file. Using defaults\n")
		return config
	}

	file, err := os.Open(fname)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	re1, err := regexp.Compile(`^(.*)=(.*)$`)
	for scanner.Scan() {
		text := scanner.Text()
		result := re1.FindStringSubmatch(text)
		config[result[1]] = result[2]
	}
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
	return config
}

// Retrieve card prices and AA card info to prime the collection pump
func getCardPriceInfo() {
	//  Retrieve from http://doc-x.net/hex/all_prices_with_uuids.txt
	//  Format:
	//    Card Name ... UUID ... Avg_price PLATINUM [# of Auctions] ... Avg_price GOLD [# of Auctions]
	//  Example:
	//    Adamanthain Scrivener ... d2222e6c-c8f8-4dad-b6dl-c0aacd3fc8f0 ...  4 PLATINUM [23 Auctions] ... 187 GOLD [231 Auctions]
	fmt.Printf("Retrieving prices from %v\n", Config["price_url"])
	resp, err := http.Get(Config["price_url"])
	if err != nil {
		fmt.Printf("Could not retrive information from price_url: '%v'. Encountered the following error: %v\n", Config["price_url"], err)
		fmt.Printf("exiting since we kinda need that information\n")
		os.Exit(1)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	s := string(body)
	lines := strings.Split(s, "\n")
	re, err := regexp.Compile(`^(.*?) \.\.\. ([\d\w-]+) \.\.\. (\d+) PLATINUM.* \.\.\. (\d+) GOLD`)
	// We skip the first line since it's a header line
	for _, line := range lines[1:] {
		result := re.FindStringSubmatch(line)
		if len(result) > 0 {
			name := result[1]
			uuid := result[2]
			plat, _ := strconv.Atoi(result[3])
			gold, _ := strconv.Atoi(result[4])
			fmt.Printf("Name: %v - %v - %vp, %vg\n", name, uuid, plat, gold)
			// We test to see if this uuid is already in our collection. If so, update it
			if _, ok := cardCollection[uuid]; ok {
				// We can't update directly, so we create a new card, modify it's values, then reassign it back to
				// to cardCollection
				c := cardCollection[uuid]
				c.name = name
				c.uuid = uuid
				c.plat = plat
				c.gold = gold
				cardCollection[uuid] = c
			} else {
				// If it doesn't exist, create a new card with appropriate values and add it to the map
				c := Card{name: name, uuid: uuid, plat: plat, gold: gold}
				cardCollection[uuid] = c
			}
		}
	}
}

func main() {
	// Read config file
	Config = loadDefaults()
	Config = readConfig("config.ini", Config)
	fmt.Printf("Using the following configuration values\n\tPrice URL (price_url): '%v'\n\tCollection file (collection_file): '%v'\n\tAlternate Art/Promo List URL(aa_promo_url): '%v'\n", Config["price_url"], Config["collection_file"], Config["aa_promo_url"])
	// Retrieve card price info
	getCardPriceInfo()
	// Register http handlers before starting the server
	http.HandleFunc("/", incoming)
	// Now that we've registered what we want, start it up
	log.Fatal(http.ListenAndServe(":5000", nil))
}

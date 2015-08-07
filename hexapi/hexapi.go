package main

//  Things needed to do
//  + Parse JSON into something I can do stuff with
//  + Read in config values from a config file
//  + Get price data from URL
//  - Handle the following types of events
//    + Login
//    + Logout
//    + DraftPack
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

type nameToUUIDMap map[string]string

var ntum nameToUUIDMap = make(nameToUUIDMap)

type sortedCardCollection struct {
	cc map[string]Card
	s  []string
}

var scc sortedCardCollection

// Configuration values we'll use all around
var Config = make(map[string]string)

// And some general variables we'll use to keep track of things
var GameStartTime time.Time = time.Now()
var packValue int = 0
var packCost int = 0
var draftCardsPicked map[string]int = make(map[string]int)

// FUNCTIONS

// Something to print out details of our collection for all cards we have at least 1 of
func printCollection() {
	// Make map of names to uuids
	for _, c := range cardCollection {
		k := c.name
		ntum[k] = c.uuid
	}
	// Make array of the keys of that map
	nmk := make([]string, len(ntum))
	// Populate that array
	n := 0
	for k, _ := range ntum {
		nmk[n] = k
		n++
	}
	// Sort that array
	sort.Strings(nmk)
	// Then use that array to print out card info in Alphabetical order
	for _, name := range nmk {
		uuid := ntum[name]
		entry := cardCollection[uuid]
		if entry.qty > 0 {
			printCardInfo(entry)
		}
	}
}

func cardUpdatedEvent(f map[string]interface{}) {
	fmt.Printf("CardUpdatedEvent: %v\n", f)
}

// Modify card quantities
func incrementCardCount(uuid string) {
	changeCardCount(uuid, 1)
}
func decrementCardCount(uuid string) {
	changeCardCount(uuid, -1)
}
func changeCardCount(uuid string, i int) {
	// Check to see if we've got an entry in draftCardsPicked we need to account for
	if _, ok := draftCardsPicked[uuid]; ok {
		if draftCardsPicked[uuid] > 0 {
			decrementDraftCardsPicked(uuid)
			return
		}
	}
	if _, ok := cardCollection[uuid]; ok {
		c := cardCollection[uuid]
		c.qty += i
		fmt.Printf("New qty for '%v' is %v (modified by %v)\n", c.name, c.qty, i)
		cardCollection[uuid] = c
	}
}
func incrementDraftCardsPicked(uuid string) {
	changeDraftCardsCount(uuid, 1)
}
func decrementDraftCardsPicked(uuid string) {
	changeDraftCardsCount(uuid, -1)
}
func changeDraftCardsCount(uuid string, i int) {
	if _, ok := draftCardsPicked[uuid]; ok {
		draftCardsPicked[uuid] += i
	} else {
		draftCardsPicked[uuid] = 0
		draftCardsPicked[uuid] += i
	}
	// Make sure we don't go negative
	if draftCardsPicked[uuid] < 0 {
		draftCardsPicked[uuid] = 0
	}
}

// Show us relevant info about the cards
func printCardInfo(c Card) {
	s := getCardInfo(c)
	fmt.Printf("%v\n", s)
}
func getCardInfo(c Card) string {
	return fmt.Sprintf("'%v' [Qty: %v] - %vp and %vg", c.name, c.qty, c.plat, c.gold)
}

// Return the UUID from a big of JSON
func getCardUUID(f map[string]interface{}) string {
	uuid_map := f["Guid"].(map[string]interface{})
	uuid := uuid_map["m_Guid"].(string)
	return uuid
}

// Handle picking of Draft Cards
// Immediately increment the count of the card, but also keep track of this
// so we can adjust later when the Collection Update events come in
func draftCardPickedEvent(f map[string]interface{}) {
	card := f["Card"].(map[string]interface{})
	uuid := getCardUUID(card)
	incrementCardCount(uuid)
	incrementDraftCardsPicked(uuid)
}

// Process draft pack choices
func draftPackEvent(f map[string]interface{}) {
	haveLeastOf := Card{name: "bogusvalue"}
	worthMostGold := Card{name: "bogusvalue"}
	worthMostPlat := Card{name: "bogusvalue"}
	cards, _ := f["Cards"].([]interface{})
	if len(cards) == 15 {
		packValue = 0
	}
	for _, u := range cards {
		card := u.(map[string]interface{})
		uuid := getCardUUID(card)
		c := cardCollection[uuid]
		// If this is the first time through, these will be nil
		if haveLeastOf.name == "bogusvalue" {
			haveLeastOf = c
		}
		if worthMostGold.name == "bogusvalue" {
			worthMostGold = c
		}
		if worthMostPlat.name == "bogusvalue" {
			worthMostPlat = c
		}
		haveLeastOf = leastQty(haveLeastOf, c)
		worthMostGold = mostGold(worthMostGold, c)
		worthMostPlat = mostPlat(worthMostPlat, c)
	}
	mostGold := getCardInfo(worthMostGold)
	mostPlat := getCardInfo(worthMostPlat)
	haveLeast := getCardInfo(haveLeastOf)
	fmt.Printf("Worth most gold: %v\n", mostGold)
	fmt.Printf("Worth most plat: %v\n", mostPlat)
	fmt.Printf("Have least of: %v\n", haveLeast)
	// Now that we've done the comparison, print out our ROI for the pack
	if len(cards) == 1 {
		packValue += worthMostPlat.plat
		fmt.Println("Total pack value: %v and pack profit is %v", packValue, packValue-packCost)
	}
}

// Comparison functions between cards
// We use '>' and '<' to favor cards of higher rarity if/when there's a tie since they show up later
// in the packs (as they are right now)
func leastQty(c1, c2 Card) Card {
	if c1.qty < c2.qty {
		return c1
	}
	return c2
}

func mostGold(c1, c2 Card) Card {
	if c1.gold > c2.gold {
		return c1
	}
	return c2
}

func mostPlat(c1, c2 Card) Card {
	if c1.plat > c2.plat {
		return c1
	}
	return c2
}

// Process Collection Event
func collectionEvent(f map[string]interface{}) {
	action := f["Action"]
	added, _ := f["CardsAdded"].([]interface{})
	removed, _ := f["CardsRemoved"].([]interface{})
	if action == "Overwrite" {
		fmt.Printf("Got an Overwrite Collection message\n")
		// First thing we do is reset counts on all cards
		for k, v := range cardCollection {
			v.qty = 0
			cardCollection[k] = v
		}
	}
	// Ok, let's extract the cards and update the numbers of each card.
	// TODO: cache that locally (in case we need to restart for some reason)
	for _, u := range added {
		card := u.(map[string]interface{})
		uuid := getCardUUID(card)
		incrementCardCount(uuid)
	}
	for _, u := range removed {
		card := u.(map[string]interface{})
		uuid := getCardUUID(card)
		decrementCardCount(uuid)
	}
	// For DEBUGGING
	//printCollection()
}

// Message: {"Winners":["Uzume, Grand Concubunny"],"Losers":["Warmaster Fuzzuko"],"User":"InGameName","Message":"GameEnded"}
func gameEndedEvent(f map[string]interface{}) {
	elapsed := time.Since(GameStartTime)
	winners := f["Winners"].([]interface{})
	losers := f["Losers"].([]interface{})
	fmt.Printf("%v triumphed over %v in an elapsed time of %s\n", winners[0], losers[0], elapsed)
}

// Message: {"Players":[],"User":"InGameName","Message":"GameStarted"}
func gameStartedEvent() {
	GameStartTime = time.Now()
	fmt.Printf("Game started at %v\n", GameStartTime)
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
	//	fmt.Println("REPLY BODY: ", string(body))
	//err = json.Unmarshal(body, &t)
	var f map[string]interface{}
	err = json.Unmarshal(body, &f)
	if err != nil {
		panic("AIEEE: Could not Unmarshall the body")
	}
	//	fmt.Println("DEBUG: Unmarshall successful")
	//	fmt.Println("DEBUG: Message is", f["Message"])
	msg := f["Message"]
	switch msg {
	case "CardUpdated":
		//		fmt.Printf("Got a Card Updated message\n")
		cardUpdatedEvent(f)
	case "Collection":
		//		fmt.Printf("Got a Collection message\n")
		collectionEvent(f)
	case "DraftCardPicked":
		//		fmt.Printf("Got a Draft Card Picked message\n")
		draftCardPickedEvent(f)
	case "DraftPack":
		//		fmt.Printf("Got a Draft Pack message\n")
		draftPackEvent(f)
	case "GameEnded":
		//		fmt.Printf("Got a Game Ended message\n")
		gameEndedEvent(f)
	case "GameStarted":
		//		fmt.Printf("Got a Game Started message\n")
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
	fmt.Println("Processing price data")
	// We skip the first line since it's a header line
	for _, line := range lines[1:] {
		result := re.FindStringSubmatch(line)
		if len(result) > 0 {
			name := result[1]
			uuid := result[2]
			plat, _ := strconv.Atoi(result[3])
			gold, _ := strconv.Atoi(result[4])
			//			fmt.Printf("Name: %v - %v - %vp, %vg\n", name, uuid, plat, gold)
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
				// And update our name to uuid map
				ntum[name] = uuid
			}
		}
	}
	// Grab out the draft pack price here
	draftPack := cardCollection["draftpak-0000-0000-0000-000000000000"]
	packCost = draftPack.plat
	// And now let them know we're ready
	fmt.Println("Beginning to listen for API events")
}

func main() {
	// Read config file
	Config = loadDefaults()
	Config = readConfig("config.ini", Config)
	//	fmt.Printf("Using the following configuration values\n\tPrice URL (price_url): '%v'\n\tCollection file (collection_file): '%v'\n\tAlternate Art/Promo List URL(aa_promo_url): '%v'\n", Config["price_url"], Config["collection_file"], Config["aa_promo_url"])
	// Retrieve card price info
	getCardPriceInfo()
	// Register http handlers before starting the server
	http.HandleFunc("/", incoming)
	// Now that we've registered what we want, start it up
	log.Fatal(http.ListenAndServe(":5000", nil))
}

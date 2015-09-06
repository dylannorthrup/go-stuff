package main

//  Things needed to do
//  + Parse JSON into something I can do stuff with
//  + Read in config values from a config file
//  + Get price data from URL
// 	+ Profit and loss for draft packs
// 	+ Better sorting/comparisons for picking purposes when items are equal
//  - Handle the following types of events
//    + Login
//    + Logout
//    + DraftPack
// 		+ Collection
// 		- SaveDeck
//    + DraftCardPicked/DaraftCardPicked
//    + GameStarted
//    + GameEnded
//    - PlayerUpdated
//    * CardUpdated (Some handled, but more work can eb done)
//  + Track pack data and indicate which cards were picked when packs wheel
//  + Track Profit/Loss for drafts
//  + Check for updated versions by checking remote URL
//  + Configurable version URLs
//  + Detect and Ignore duplicate API messages
//
//  Stretch Goals
//  - Post card data to remote URL (for collating draw data)

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Card The Cards we work with and all the info we need about them
type Card struct {
	name   string
	uuid   string
	qty    int
	rarity string
	gold   int
	plat   int
}

// The Version of the program so we can figure out if we're using the most recent version
var programVersion = "0.3"

// Vars so we can figure out what our update URL is
var programName = os.Args[0]
var programPlatform = runtime.GOOS
var programArch = runtime.GOARCH

// And this is all our cards together... we use uuid as a key
var cardCollection = make(map[string]Card)

// We use this for sorting by name instead of by UUID. ntum = "Name To UUID Map"
var ntum = make(map[string]string)

// Configuration values we'll use all around
var Config = make(map[string]string)

// And some general variables we'll use to keep track of things
var GameStartTime = time.Now()

var packValue int
var packCost int
var packNum int
var packContents [16]string
var previousContents [16]string
var draftCardsPicked = make(map[string]int)
var sessionProfit int
var lastAPIMessage string

var loadingCacheOrPriceData = false
var currentlyDrafting = false

// Refresh price data every two hours
var priceRefreshTimerPeriod = time.Hour * time.Duration(2)
var priceRefreshCacheTimer *time.Timer

// Cache collection data 20 seconds after the last collection message was received
var collectionTimerPeriod = time.Second * time.Duration(20)
var collectionCacheTimer *time.Timer

// List of locations cards can be for CardUpdated events
var cardCollectionMap = map[string]string{
	"0":   "Champion",
	"1":   "Deck",
	"2":   "Hand",
	"4":   "Opposing Champion",
	"8":   "Play",
	"16":  "Discard",
	"32":  "Void",
	"64":  "Shard",
	"128": "Chain",
	"256": "Tunnelled",
	"512": "Choose Effect",
}

// FUNCTIONS

// Something to print out details of our collection for all cards we have at least 1 of
func printCollection() {
	// Get sorted array of card names
	nmk := listCardsSortedByName()
	// Then use that array to print out card info in Alphabetical order
	for _, name := range nmk {
		uuid := ntum[name]
		entry := cardCollection[uuid]
		if entry.qty > 0 {
			printCardInfo(entry)
		}
	}
}

// Guesses about CardUpdated flags importance
// Collection is a power of 2 map. They refer to different "zones". Here's what I think they
// correspond to:
//	0 - Champion
//	1 - Deck
//	2 - Hand
//	4 - Opposing Champion
//	8 - In Play (battle zone)
//	16 - Discard Pile
//	32 - Void
//	64 - Shard Zone (where shards go when they're played)
//	128 - The Chain
//	256 - Tunnelled
//  512 - Effect Choose???
//
// State
//	8192 - Ready
//	16384 - Ready on opponent's turn?
//	16517 - Exhausted
//	16513 - Exhausted on opponent's turn?
func cardUpdatedEvent(f map[string]interface{}) {
	/* DEBUGGING TO LEARN WHAT'S UP WITH THE FLAGS
	keys := []string{}
	for k := range f {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if strings.Contains(k, "Abilities") ||
			strings.Contains(k, "BaseTemplate") ||
			strings.Contains(k, "Message") ||
			strings.Contains(k, "User") {
			// fmt.Println("\nSKIPPING Abilities section")
			continue
		}
		fmt.Printf("%v: %v - ", k, f[k])
	}
	fmt.Println("")
	/*	END DEBUGGING TO LEARN WHAT'S UP WITH THE FLAGS */
	name := f["Name"].(string)
	if name == "" {
		return
	}
	stats := fmt.Sprintf("[(%v/%v) for %v]", f["Attack"], f["Defense"], f["Cost"])
	collection := fmt.Sprintf("%v", f["Collection"])
	if collection == "8" || collection == "16" || collection == "256" {
		return
	}
	fmt.Printf("In %v Zone:\t'%v' %v\n", cardCollectionMap[collection], name, stats)
	// shards, _ := f["Shards"].(int)
	// attrs, _ := f["Attributes"].(int)
	// collection, _ := f["Collection"].(int)
	// state, _ := f["State"].(int)
	// fmt.Printf("CardUpdatedEvent: '%v' state: %v, shards: %v, attrs: %v, collection: %v\n", name, state, shards, attrs, collection)
	// fmt.Printf(" - %v\n", f)
}

// Modify card quantities
func incrementCardCount(uuid string) {
	changeCardCount(uuid, 1)
}
func decrementCardCount(uuid string) {
	changeCardCount(uuid, -1)
}
func setCardCount(uuid string, i int) {
	if _, ok := cardCollection[uuid]; ok {
		c := cardCollection[uuid]
		c.qty = 0
		changeCardCount(uuid, i)
	}
}
func changeCardCount(uuid string, i int) {
	// fmt.Println("Inside changeCardCount()")
	// If we're currently drafting, we want to skip this check (it'll all come out in the wash later, but
	// for now we want these counts to go up so they're accurately reflected in the selection output)
	if !currentlyDrafting {
		// Check to see if we've got an entry in draftCardsPicked we need to account for
		if _, ok := draftCardsPicked[uuid]; ok {
			if draftCardsPicked[uuid] > 0 {
				// c := cardCollection[uuid]
				// fmt.Printf("INFO: Not incrementing %v because it's on the draft list %v times\n", c.name, draftCardsPicked[uuid])
				decrementDraftCardsPicked(uuid)
				// fmt.Println("About to return from inside changeCardCount")
				return
			}
		}
	}
	// fmt.Println("No draft cards for this type. Handing off to rawChangeCardCount")
	// if not, go ahead and call the dangerous function
	rawChangeCardCount(uuid, i)
}
func rawChangeCardCount(uuid string, i int) {
	if _, ok := cardCollection[uuid]; ok {
		c := cardCollection[uuid]
		c.qty += i
		if loadingCacheOrPriceData == false {
			fmt.Printf("INFO: New collection qty for '%v' is %v (modified by %v)\n", c.name, c.qty, i)
		}
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
		// fmt.Printf("INFO: Bumping %v by %v for total of %v\n", uuid, i, draftCardsPicked[uuid])
		draftCardsPicked[uuid] += i
	} else {
		// fmt.Printf("INFO: Adding %v to map and setting value to 1\n", uuid)
		draftCardsPicked[uuid] = 0
		draftCardsPicked[uuid] += i
	}
	// Make sure we don't go negative
	if draftCardsPicked[uuid] < 0 {
		// fmt.Printf("INFO: PREVENTING us from going negative for UUID %v\n", uuid)
		draftCardsPicked[uuid] = 0
	}
	// Call the rawChangeCardCount function here so we have one stop shopping for Draft func calls
	// rawChangeCardCount(uuid, i)
}

// Check our program version to see it it's the most recent and, if not, tell
// the kind users that they should upgrade.
func checkProgramVersion() {
	fmt.Print("Checking for program updates . . . ")
	versionURL := Config["version_url"]
	resp, err := http.Get(versionURL)
	if err != nil {
		fmt.Printf("Could not retrive version information from version url: '%v'. Encountered the following error: %v\n", versionURL, err)
		return
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	version := string(body)
	version = strings.Replace(version, "\n", "", 1)
	if programVersion == version {
		fmt.Printf("Running up to date version '%v'\n", version)
		time.Sleep(time.Second)
	} else {
		downloadURL := fmt.Sprintf("http://doc-x.net/hex/downlaods/hexapi_%v_%v", programPlatform, programArch)
		if programPlatform == "windows" {
			downloadURL = fmt.Sprintf("%v.exe", downloadURL)
		}
		fmt.Print(" version mismatch. WARNING!!!\n")
		fmt.Printf("\tCurrent version is '%v'. You are running version '%v'.\n", version, programVersion)
		fmt.Printf("\tPlease download a new copy from %v\n", downloadURL)
		secondString := "seconds"
		for i := 5; i > 0; i-- {
			if i == 1 {
				secondString = "second"
			}
			fmt.Printf("\tContinuing in %v %v\n", i, secondString)
			time.Sleep(time.Second)
		}
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
	uuidMap := f["Guid"].(map[string]interface{})
	uuid := uuidMap["m_Guid"].(string)
	return uuid
}

// Handle picking of Draft Cards
// Immediately increment the count of the card, but also keep track of this
// so we can adjust later when the Collection Update events come in
func draftCardPickedEvent(f map[string]interface{}) {
	// Make sure we know we're drafting
	currentlyDrafting = true
	card := f["Card"].(map[string]interface{})
	uuid := getCardUUID(card)
	incrementCardCount(uuid)
	incrementDraftCardsPicked(uuid)
	c := cardCollection[uuid]
	info := getCardInfo(c)
	fmt.Printf("++ Pack [%v]: You Drafted %v\n", packNum, info)
	packValue += c.plat
	// Put something here to remove c.name from packContents[packNum]
	if packNum > 8 {
		prevCard := fmt.Sprintf("'%v', ", c.name)
		previousContents[packNum] = strings.Replace(previousContents[packNum], prevCard, "", 1)
	}
	if packNum == 1 {
		packProfit := packValue - packCost
		fmt.Println("==========================    PACK AND SESSION STATISTICS    ==========================")
		fmt.Printf("Total pack value: %v plat. Pack profit is %v plat and total session profit is %v plat.\n", packValue, packProfit, sessionProfit)
		fmt.Println("==========================    PACK AND SESSION STATISTICS    ==========================")

	}
	// And unset this in case we're done
	currentlyDrafting = false
}

// Process draft pack choices
func draftPackEvent(f map[string]interface{}) {
	haveLeastOf := Card{name: "bogusvalue"}
	worthMostGold := Card{name: "bogusvalue"}
	worthMostPlat := Card{name: "bogusvalue"}
	cards, _ := f["Cards"].([]interface{})
	numCards := len(cards)
	// We need this for stuff when the DraftCard event fires
	packNum = numCards
	// reset the pack value for a new pack along with all the pack tracking arrays
	if numCards == 15 {
		packValue = 0
		for n := range packContents {
			packContents[n] = ""
			previousContents[n] = ""
		}
	}
	if numCards < 8 {
		prevNum := numCards + 8
		previousContents[numCards] = packContents[prevNum]
	}
	// Do some computations to figure out the optimal picks for plat, gold and filling out our collection
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
		packContents[numCards] = fmt.Sprintf("%v, '%v'", packContents[numCards], c.name)

		// If we have 9 or more cards in pack, save what we've got so we've got so we can determine what others picked
		if numCards < 8 {
			prevCard := fmt.Sprintf("'%v', ", c.name)
			previousContents[numCards] = strings.Replace(previousContents[numCards], prevCard, "", 1)
		}
	}
	packContents[numCards] = strings.Replace(packContents[numCards], ", ", "", 1)
	fmt.Printf("== Pack [%v] Contents: %v\n", numCards, packContents[numCards])
	if numCards < 8 {
		fmt.Printf("-- MISSING CARDS: %v\n", previousContents[numCards])
	}
	mostGold := getCardInfo(worthMostGold)
	mostPlat := getCardInfo(worthMostPlat)
	haveLeast := getCardInfo(haveLeastOf)
	fmt.Println("** Computed best picks from pack:")
	fmt.Printf("\tWorth most plat: %v\n", mostPlat)
	fmt.Printf("\tWorth most gold: %v\n", mostGold)
	fmt.Printf("\tHave least of: %v\n", haveLeast)
	// Now that we've done the comparison, print out our ROI for the pack
	if numCards == 1 {
		packValue += worthMostPlat.plat
		packProfit := packValue - packCost
		sessionProfit += packProfit
		// fmt.Println("==========================    PACK AND SESSION STATISTICS    ==========================")
		// fmt.Printf("Total pack value: %v plat. Pack profit is %v plat and total session profit is %v plat.\n", packValue, packProfit, sessionProfit)
		// fmt.Println("==========================    PACK AND SESSION STATISTICS    ==========================")
	}
}

// Comparison functions between cards
// We use 'mostGold' as the ultimate tie breaker (since it's less likely to be equal than the other two)
// For 'mostGold', We use '>' and '<' to favor cards of higher rarity if/when there's a tie since they show up later
// in the packs (as they are right now)
func leastQty(c1, c2 Card) Card {
	if c1.qty == c2.qty {
		return mostPlat(c1, c2)
	}
	if c1.qty < c2.qty {
		return c1
	}
	return c2
}

func mostPlat(c1, c2 Card) Card {
	if c1.plat == c2.plat {
		return mostGold(c1, c2)
	}
	if c1.plat > c2.plat {
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

// Something we use to write out a cache of our collection
func cacheCollection() {
	// fmt.Printf("Entered cacheCollection() with timer of %v\n", collectionCacheTimer)
	// First thing we do is stop the timer
	collectionCacheTimer.Stop()
	// fmt.Printf("collectionCacheTimer stopped: %v\n", collectionCacheTimer)
	// Open file. If it exists right now, remove that before creating a new one
	cacheFile := Config["collection_file"]
	if _, err := os.Stat(cacheFile); err == nil {
		os.Remove(cacheFile)
	}
	f, err := os.Create(cacheFile)
	if err != nil {
		fmt.Printf("Could not create file %v for writing: %v\n", cacheFile, err)
		return
	}
	fmt.Printf("Caching collection info to file '%v'.  ", cacheFile)
	// Defer our close
	defer f.Close()

	// Look through cards to write out key/value data
	for k, v := range cardCollection {
		if v.qty == 0 {
			continue
		}
		line := fmt.Sprintf("%v : %v\n", k, v.qty)
		f.WriteString(line)
		// fmt.Printf(".")
	}
	f.Sync()
	// fmt.Println("! Caching of collection is complete")

	// If the user asked us to cache a CSV file, go ahead and accomodate them
	if Config["export_csv"] == "true" {
		csvFile := Config["csv_filename"]
		if _, err := os.Stat(csvFile); err == nil {
			os.Remove(csvFile)
		}
		f, err := os.Create(csvFile)
		if err != nil {
			fmt.Printf("Could not create file %v for writing: %v\n", csvFile, err)
			return
		}
		fmt.Printf("Writing CSV card data to file '%v'.\n", csvFile)
		// Defer our close
		defer f.Close()

		// Get a sorted list of cards
		nmk := listCardsSortedByName()
		// Then use that array to print out card info in Alphabetical order
		for _, name := range nmk {
			uuid := ntum[name]
			entry := cardCollection[uuid]
			if entry.qty > 0 {
				line := fmt.Sprintf("\"%v\",%v\n", entry.name, entry.qty)
				f.WriteString(line)
			}
		}
	}
}

func listCardsSortedByName() []string {
	// Make map of names to uuids
	for _, c := range cardCollection {
		k := c.name
		ntum[k] = c.uuid
	}
	// Make array of the keys of that map
	nmk := make([]string, len(ntum))
	// Populate that array
	n := 0
	for k := range ntum {
		nmk[n] = k
		n++
	}
	sort.Strings(nmk)
	return nmk
}

// Read in the info we cached up there.
func readCollectionCache() {
	cacheFile := Config["collection_file"]
	in, _ := os.Open(cacheFile)
	defer in.Close()
	scanner := bufio.NewScanner(in)
	scanner.Split(bufio.ScanLines)

	// Set this so we don't spam out card count info messages
	loadingCacheOrPriceData = true

	re1, _ := regexp.Compile(`^(.*) : (\d+)$`)
	for scanner.Scan() {
		// Split on regex and stuff count information into cardCollection
		text := scanner.Text()
		result := re1.FindStringSubmatch(text)
		uuid := result[1]
		i, _ := strconv.Atoi(result[2])
		setCardCount(uuid, i)
		// fmt.Println(scanner.Text())
	}
	// And, now that we're done, reset this
	loadingCacheOrPriceData = false
}

// Process Collection Event
func collectionEvent(f map[string]interface{}) {
	action := f["Action"]
	added, _ := f["CardsAdded"].([]interface{})
	removed, _ := f["CardsRemoved"].([]interface{})
	if action == "Overwrite" {
		fmt.Printf("Got an Overwrite Collection message. Doing full update of card collection.\n")
		// If this is an Overwrite message, first thing we do is reset counts on all cards
		for k, v := range cardCollection {
			v.qty = 0
			cardCollection[k] = v
		}
		// Also, turn off update printing to reduce spamming of the screen.
		loadingCacheOrPriceData = true
	} else if action == "Update" {
		// fmt.Print("Handling Update Collection message\n")
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
	// And, reset this (even if it wasn't set, we'll make sure it gets unset)
	loadingCacheOrPriceData = false
	// Schedule our collectionCacheTimer to write out the collection to the cache file
	// We do this because sometimes we'll get many, many, MANY updates at once. We want
	// to bundle them all up to be done in one go.
	if collectionCacheTimer != nil {
		// fmt.Printf("Stopping collectionCacheTimer '%v'\n", collectionCacheTimer)
		collectionCacheTimer.Stop()
	}
	collectionCacheTimer = time.AfterFunc(collectionTimerPeriod, cacheCollection)
	// fmt.Printf("Set new collectionCacheTimer '%v'\n", collectionCacheTimer)
	// For DEBUGGING
	// printCollection()
}

// Message: {"Winners":["Uzume, Grand Concubunny"],"Losers":["Warmaster Fuzzuko"],"User":"InGameName","Message":"GameEnded"}
func gameEndedEvent(f map[string]interface{}) {
	elapsed := time.Since(GameStartTime)
	winners := f["Winners"].([]interface{})
	winner := winners[0].(string)
	winner = strings.TrimSpace(winner)
	losers := f["Losers"].([]interface{})
	loser := losers[0].(string)      // Gotta convert this to a string
	loser = strings.TrimSpace(loser) // Then I can use TrimSpace() on it.
	fmt.Printf("%v triumphed over %v in an elapsed time of %vm %vs\n", winner, loser, int(elapsed.Minutes()), int(elapsed.Seconds())%60)
}

// Message: {"Players":[],"User":"InGameName","Message":"GameStarted"}
func gameStartedEvent() {
	GameStartTime = time.Now()
	fmt.Printf("Game started at %v\n", GameStartTime.Format(time.UnixDate))
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

func saveDeckEvent(f map[string]interface{}) {
	fmt.Println("In function of saveDeckEvent")
}

func incoming(rw http.ResponseWriter, req *http.Request) {
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		panic("AIEEE: Could not readAll for req.Body")
	}
	// Check to see if the last API message
	if lastAPIMessage == string(body) {
		// fmt.Println("INFO: Duplicate API Message. Discarding.")
		return
	}
	lastAPIMessage = string(body)
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
	case "SaveDeck":
		//		fmt.Printf("Got a Save Deckmessage\n")
		saveDeckEvent(f)
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
		//		fmt.Printf("Got a Player Updated message\n")
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
	// Do we want to export to CSV and, if so, what's the filename we want to use?
	retMap["export_csv"] = "big_fat_nope_a_rino"
	retMap["csv_filename"] = "collection.csv"
	// Default location to check for version information
	retMap["version_url"] = "http://doc-x.net/hex/downloads/hexapi_version.txt"
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
	var body []byte
	var err error
	updatingData := false
	gotHTTPError := false
	if priceRefreshCacheTimer != nil {
		updatingData = true
		fmt.Println("Updating price data to insure it is fresh")
	}
	if Config["local_price_file"] == "" {
		fmt.Printf("Retrieving prices from %v\n", Config["price_url"])
		resp, err := http.Get(Config["price_url"])
		if err != nil {
			gotHTTPError = true
		}
		// If we had a problem AND we're not simply doing an update, then exit.
		// Othwerise, continue on and we'll just be using (possibly) stale data
		if gotHTTPError && !updatingData {
			fmt.Printf("Could not retrive information from price_url: '%v'. Encountered the following error: %v\n", Config["price_url"], err)
			fmt.Printf("exiting since we kinda need that information\n")
			os.Exit(1)
		} else if gotHTTPError && updatingData {
			fmt.Println("Encountered error refreshing price data. Will try again later. Using previously cached data in the interim.")
			setPriceRefreshTimer()
			return
		}
		// If we didn't encounter a problem, read in the data so we can process it
		if !gotHTTPError {
			defer resp.Body.Close()
			body, err = ioutil.ReadAll(resp.Body)
			if err != nil {
				log.Fatal(err)
			}
		}
	} else {
		fmt.Printf("Retrieving prices from %v\n", Config["local_price_file"])
		body, err = ioutil.ReadFile(Config["local_price_file"])
		if err != nil {
			log.Fatal(err)
		}
	}
	// If we didn't have a problem retrieving the HTTP data, go ahead and process it
	// Also, this shouldn't get set if we're reading in from a local file
	if !gotHTTPError {
		s := string(body)
		lines := strings.Split(s, "\n")
		re, _ := regexp.Compile(`^(.*?) \.\.\. ([\d\w-]+) \.\.\. (\d+) PLATINUM.* \.\.\. (\d+) GOLD`)
		fmt.Println("Processing price data")

		// Reduce the spamminess of loading collection info
		loadingCacheOrPriceData = true

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
		// Now, turn back on info messages for changes in card counts
		loadingCacheOrPriceData = false
	}

	// Set our refresh timer to come back and do this again later
	setPriceRefreshTimer()

	// If we're not simply doing an update, grab out the draft pack price here
	if !updatingData {
		draftPack := cardCollection["draftpak-0000-0000-0000-000000000000"]
		packCost = draftPack.plat
	}
	// And now let them know we're ready
	fmt.Println("Price data processed")
}

func setPriceRefreshTimer() {
	// Set up a timer to refresh this information in a bit to mitigate against stale price data
	if priceRefreshCacheTimer != nil {
		// fmt.Printf("Stopping priceRefreshCacheTimer '%v'\n", priceRefreshCacheTimer)
		priceRefreshCacheTimer.Stop()
	}
	priceRefreshCacheTimer = time.AfterFunc(priceRefreshTimerPeriod, getCardPriceInfo)

}

func main() {
	// Read config file
	Config = loadDefaults()
	Config = readConfig("config.ini", Config)
	//	fmt.Printf("Using the following configuration values\n\tPrice URL (price_url): '%v'\n\tCollection file (collection_file): '%v'\n\tAlternate Art/Promo List URL(aa_promo_url): '%v'\n", Config["price_url"], Config["collection_file"], Config["aa_promo_url"])
	// Check to see if we're running the most recent version
	checkProgramVersion()
	// Retrieve card price info
	getCardPriceInfo()
	// Read in our collection cache
	readCollectionCache()
	fmt.Println("Beginning to listen for API events")
	// Register http handlers before starting the server
	http.HandleFunc("/", incoming)
	// Now that we've registered what we want, start it up
	log.Fatal(http.ListenAndServe(":5000", nil))
}

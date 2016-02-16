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
//  + Deal with missing card info in price/UUID download
//  + Print out details of individual cards in pack contents output
//  + Print out Gold and Plat value of collections
//  - Do deck summaries on Save Deck event
//	- Handle CardUpdated with ExtendedDart attributes
//
//  Stretch Goals
//  - Post card data to remote URL (for collating draw data)

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math"
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
	eaqty  int
	rarity string
	gold   int
	plat   int
}

// Player variable that we'll be using in tracking game state
type Player struct {
	name      string
	health    int
	id        int
	resources int
	blood     int
	sapphire  int
	wild      int
	diamond   int
	ruby      int
	champion  gameCard
	cards     map[string]gameCard
}

// Card in game that we're tracking
type gameCard struct {
	controller int
	cost       int
	atk        int
	def        int
	name       string
	state      int
	location   int
}

// Game variable to track game state
type Game struct {
	pnums map[string]int
	p1    Player
	p2    Player
}

var currentGame Game

// The Version of the program so we can figure out if we're using the most recent version
var programVersion = "0.9"

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
var packGoldValue int
var collectionGoldValue = 0
var collectionPlatValue = 0
var packCost int
var packGoldCost int
var goldPlatRatio int // How many gold for a single plat
var packNum int
var packContents [18]string
var previousContents [18]string
var draftCardsPicked = make(map[string]int)
var sessionPlatProfit int
var sessionGoldProfit int
var lastAPIMessage string

var loadingCacheOrPriceData = false
var currentlyDrafting = false

var packSize = 17

// Refresh price data every two hours
var priceRefreshTimerPeriod = time.Hour * time.Duration(2)
var priceRefreshCacheTimer *time.Timer

// Cache collection data 20 seconds after the last collection message was received
var collectionTimerPeriod = time.Second * time.Duration(20)
var collectionCacheTimer *time.Timer

// List of locations cards can be for CardUpdated events
var cardCollectionMap = map[int]string{
	0:   "Champion",
	1:   "Deck",
	2:   "Hand",
	4:   "Opposing Champion",
	8:   "Play",
	16:  "Discard",
	32:  "Void",
	64:  "Shard",
	128: "Chain",
	256: "Tunnelled",
	512: "Choose Effect",
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
// Collection is a power of 2 map. They refer to different "zones". Here's what they correspond to:
//	0 - None
//	1 - Deck
//	2 - Hand
//	4 - Champions
//	8 - In Play (battle zone)
//	16 - Discard Pile
//	32 - Void
//	64 - Played Resources
//	128 - The Chain
//	256 - Underground
//  512 - Effect Choose
//	1024 - Mod
//
// Card Attributes
//	1 - 0 - Lifedrain
//	2 - 1 - Flight
//	4 - 2 - Speed
//	8 - 3 - Skyguard
//	16 - 4 - Crush
//	32 - 5 - Steadfast
//	64 - 6 - Invincible
//	128 - 7 - Spellshield
//	256 - 8 - Unique
//	512 - 9 - Can't Attack
//	1024 - 10 - Can't Block
//	2048 - 11 - Defensive
//	4096 - 12 - Must Attack
//	8192 - 13 - Does not auto-ready
//	16384 - 14 -  Swiftstrike
//	32768 - 15 - Rage
//	65536 - 16 - Must Block
//	131072 - 17 - Unblockable
// 	- 18 - Prevent Combat Damage
//	- 19 - Prevent Non-Combat Damage (also, 2^18 + 2^19 = prevent all damage)
// 	- 20 - Unimplemented (Doublestrike)
//	- 21 - Cannot Inflict Combat Damage
//	- 22 - Cannot Inflict Non-Combat Damage (also, 2^21 + 2^22 = Cannot inflict damage)
//	- 23 - Enters Play Exhausted
//	- 24 - Inspire
//	- 25 - Escalation
//	- 26 - Does not ready next ready step
//	- 27 - Lethal, but voids damaged troops
//	- 28 - Quick
//	- 29 - Blessing of the Fallen (can Inspire from Crypt)
//	- 30 - Must be blocked
//

// Card States
//	1 - 0 - None
//	2 - 1 - Exhausted
//	4 - 2 - Blocking
//	8	- 3 - Attacking
//	16 - 4 - Damaged
//	32 - 5 - Healed
//	64 - 6 - Dead
//	128	- 7 - Has Attacked
//	256	- 8 - Has Blocked
//	512 - 9 - Effect Expired
//	1024 - 10 - Zone Change Replacement
//	2048 - 11 - Activated
//	4096 - 12 - Voids if Destroyed
//	8192 - 13 - Came out this turn
//	16384 - 14 - Started a turn on your side
//
var cardStates = []string{
	"None",
	"exhausted",
	"blocking",
	"attacking",
	"damaged",
	"healed",
	"dead",
	"has attacked",
	"has blocked",
	"effect expired",
	"zone change replacement",
	"activated",
	"voids if destroyed",
	"came out this turn",
	"started the turn on your side",
}

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
	atk := floatToInt(f["Attack"].(float64))
	def := floatToInt(f["Defense"].(float64))
	cost := floatToInt(f["Cost"].(float64))
	state := floatToInt(f["State"].(float64))
	shards := f["Shards"]
	attrs := f["Attributes"]
	stats := fmt.Sprintf("[(%v/%v) for %v]", atk, def, cost)
	collection, _ := strconv.Atoi(fmt.Sprintf("%v", f["Collection"]))
	controller := floatToInt(f["Controller"].(float64))
	var player *Player

	// Figure out which player controls this card
	if currentGame.p1.id == controller {
		// fmt.Printf("controller: %v and id: %v\n", controller, currentGame.p1.id)
		player = &currentGame.p1
	} else if currentGame.p2.id == controller {
		player = &currentGame.p2
	} else if currentGame.p2.id == 2 {
		// See if this player is config'd and figure out if it's p1 or p2 and set it appropriately if need be
		_ = checkPlayerConfigured(controller)
	} else {
		fmt.Printf("Could not find player with controller number of %v\n", controller)
		showGameState()
		return
	}

	// fmt.Printf("Current Player Champion name: %v\n", player.champion.name)
	// Do a thing here to match this message with a card we know is in the game already.  Based on that, we can
	// either create a new card or update the old card

	switch collection {
	case 1: // Deck Zone
		{
			fmt.Printf("DECK: %v's %v was sent back into their deck\n", player.name, name)
			return
		}
	case 2: // Hand Zone
		{
			fmt.Printf("HAND: %v's %v went into their hand\n", player.name, name)
			return
		}
	case 4: // Champion Zone
		{
			if Config["debug_cardUpdated"] == "true" {
				fmt.Printf("CCC= Working on %v with def %v, state %v and collection %v for player %v\n", name, def, translateCardState(state), collection, player)
			}
			if player == nil {
				// TODO: Deal with the nil player gracefully. Until then, just let this get populated by a later message
				// player.champion = gameCard{def: def, name: name, state: state, location: 8}
				// // name, atk, def, state, attrs
				// fmt.Printf("Champion %v starting off with %v health and state of %v\n", name, def, translateCardState(state))
				return
			}
			if player.champion.name == "Unknown" {
				player.champion.name = name
				return
			}
			champ := player.champion
			// if champ.def != def || champ.state != state {
			if champ.def != def {
				modification := "lost"
				if champ.def < def {
					modification = "gained"
				}
				difference := champ.def - def
				if difference < 0 {
					difference = difference * -1
				}
				fmt.Printf("CHAMP: %v %v %v health and now has %v health\n", name, modification, difference, def)
				// fmt.Printf("health %v and state %v\n", def, translateCardState(state))
				// fmt.Printf("Champion %v now has health %v and state %v\n", name, def, state)
				player.champion.def = def
				// TODO: Once we have a better idea about the states, pull this out into it's own block
				player.champion.state = state
			} else {
				// fmt.Printf("\tCHAMP: No change for %v\n", name)
			}
		}
		return
	case 8:
		{
			// fmt.Printf("CardUpdatedEvent: '%v' state: %v, shards: %v, attrs: %v, collection: %v\n", name, state, shards, attrs, collection)
			fmt.Printf("\tBATTLE: %v's %v [%v/%v] state: %v; shards: %v; attrs: %v\n", player.name, name, atk, def, translateCardState(state), shards, attrs)
			return
		}
	case 16:
		{
			fmt.Printf("\tDISCARD: %v's %v was sent to their discard pile\n", player.name, name)
			return
		}
	case 32:
		{
			fmt.Printf("\tVOID: %v's %v was sent to the void\n", player.name, name)
			return
		}
	case 64:
		{
			fmt.Printf("\tSHARD: %v played %v as a resource\n", player.name, name)
			return
		}
	case 128:
		{
			fmt.Printf("\tCHAIN: %v played %v\n", player.name, name)
			return
		}
	case 256:
		{
			fmt.Printf("\tUNDERGROUND: %v tunnelled %v\n", player.name, name)
			return
		}
	default:
		{
			fmt.Printf("In %v Zone:\t'%v' %v\n", cardCollectionMap[collection], name, stats)
			return
		}
	}
	// if collection == "8" || collection == "16" || collection == "256" {
	// 	return
	// }

	// shards, _ := f["Shards"].(int)
	// attrs, _ := f["Attributes"].(int)
	// collection, _ := f["Collection"].(int)
	// state, _ := f["State"].(int)
	// fmt.Printf("CardUpdatedEvent: '%v' state: %v, shards: %v, attrs: %v, collection: %v\n", name, state, shards, attrs, collection)
	// fmt.Printf(" - %v\n", f)
}

func translateCardState(state int) string {
	// Init this to an empty string
	returnString := ""
	// Shortcut for state of 0
	if state == 0 {
		return "None"
	}
	// Set this to the top value we'd want to be looking at which is 2^(length of CardStates)
	checkingStateValue := len(cardStates)

	// Count down the possible states and, if it's there, add it to the return string
	for true {
		if state >= checkingStateValue {
			// Figure out the index of the cardStates array we need to refer to
			index := intLog2(checkingStateValue)
			// If the index is 0, go ahead and return the returnString
			if index <= 0 {
				return returnString
			}
			// Otherwise, go ahead and process things
			if Config["debug_card_states"] == "true" {
				fmt.Printf("**** Index: %v from checkingStateValue %v\n", index, checkingStateValue)
			}
			// Some checking here so we can have nice formatting
			if returnString == "" {
				returnString = cardStates[index]
			} else {
				returnString = fmt.Sprintf("%v, %v", returnString, cardStates[index])
			}
			// And, remove the current checkingStateValue from the state so we count down
			state -= checkingStateValue
		}
		checkingStateValue = checkingStateValue / 2
	}
	return returnString
}

// Do an integer version of math.Log2.  Make sure it *IS* a power of two, then
func intLog2(num int) int {
	floatNum := float64(num)
	power := math.Log2(floatNum)
	if math.Mod(power, 1.0) != 0.0 {
		power, _ = math.Modf(power)
	}
	intPower := int(power)
	return intPower
}

// Dump out current game state. For debugging. Shouldn't see this in normal operations
func showGameState() {
	fmt.Printf("!!!showGameState: pnums: %v\n", currentGame.pnums)
	fmt.Printf("!!!showGameState: p1: %v\n", currentGame.p1)
	fmt.Printf("!!!showGameState: p2: %v\n", currentGame.p2)
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
	if Config["debug_collection_update"] == "true" {
		c := cardCollection[uuid]
		fmt.Printf("INFO: [changeCardCount] Changing qty for '%v' by %v (new qty %v)\n", c.name, i, c.qty)
	}
	rawChangeCardCount(uuid, i)
}

func rawChangeCardCount(uuid string, i int) {
	// If the card is in the system, update it
	if _, ok := cardCollection[uuid]; ok {
		c := cardCollection[uuid]
		c.qty += i
		if loadingCacheOrPriceData == false || Config["debug_collection_update"] == "true" {
			fmt.Printf("INFO: New collection qty for '%v' is %v (modified by %v)\n", c.name, c.qty, i)
		}
		cardCollection[uuid] = c
	} else {
		if loadingCacheOrPriceData == false || Config["debug_collection_update"] == "true" {
			fmt.Printf("INFO: No card with UUID of %v exists. Cannot change its quantity\n.", uuid)
		}
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
	if Config["detailed_card_info"] == "true" {
		return fmt.Sprintf("'[%v] %v' %v [Qty: %v] - %vp and %vg", c.rarity, c.name, c.uuid, c.qty, c.plat, c.gold)
	}
	return fmt.Sprintf("'%v' [Qty: %v] - %vp and %vg", c.name, c.qty, c.plat, c.gold)
}

func getCardCount(uuid string) int {
	return cardCollection[uuid].qty
}

// Return the UUID from a big pile of JSON
func getCardUUIDFromJSON(f map[string]interface{}) string {
	uuidMap := f["Guid"].(map[string]interface{})
	uuid := uuidMap["m_Guid"].(string)
	return uuid
}

// Return the Card name from a big pile of JSON
func getCardNameFromJSON(f map[string]interface{}) string {
	// uuidMap := f["Guid"].(map[string]interface{})
	name := f["Name"].(string)
	return name
}

// Handle picking of Draft Cards
// Immediately increment the count of the card, but also keep track of this
// so we can adjust later when the Collection Update events come in
func draftCardPickedEvent(f map[string]interface{}) {
	// Make sure we know we're drafting
	currentlyDrafting = true
	card := f["Card"].(map[string]interface{})
	uuid := getCardUUIDFromJSON(card)
	incrementCardCount(uuid)
	incrementDraftCardsPicked(uuid)
	c := cardCollection[uuid]
	info := getCardInfo(c)
	fmt.Printf("++ Pack [%v]: You Drafted %v\n", packNum, info)
	if Config["debug_pack_value"] == "true" {
		fmt.Printf("==== DEBUG: [DraftCardPickedEvent] Adding %v to current pack value of %v (should total %v)\n", c.plat, packValue, c.plat+packValue)
	}

	packValue += c.plat
	packGoldValue += c.gold
	// Put something here to remove c.name from packContents[packNum]
	if packNum > 8 {
		prevCard := fmt.Sprintf("'%v', ", c.name)
		previousContents[packNum] = strings.Replace(previousContents[packNum], prevCard, "", 1)
	}
	if packNum == 1 {
		if Config["debug_pack_value"] == "true" {
			fmt.Printf("==== DEBUG: [DraftCardPickedEvent] Session Plat profit prior to modification: %v\n", sessionPlatProfit)
			fmt.Printf("==== DEBUG: [DraftCardPickedEvent] Session Gold profit prior to modification: %v\n", sessionGoldProfit)
		}
		packProfit := packValue - packCost
		packGoldProfit := packGoldValue - packGoldCost
		sessionPlatProfit += packProfit
		sessionGoldProfit += packGoldProfit
		if Config["debug_pack_value"] == "true" {
			fmt.Printf("==== DEBUG: [DraftCardPickedEvent] Session profit after modification: %v (pack value of %v and pack cost of %v)\n", sessionPlatProfit, packValue, packCost)
			fmt.Printf("==== DEBUG: [DraftCardPickedEvent] Session profit after modification: %v (pack value of %v and pack cost of %v)\n", sessionGoldProfit, packGoldValue, packGoldCost)
		}
		fmt.Println("==========================    PACK AND SESSION STATISTICS    ==========================")
		fmt.Printf("Total pack value: %v plat (%v gold). Pack profit is %vp (%vg) and total session profit is %vp (%vg).\n", packValue, packGoldValue, packProfit, packGoldProfit, sessionPlatProfit, sessionGoldProfit)
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
	if numCards == packSize {
		packValue = 0
		for n := range packContents {
			packContents[n] = ""
			previousContents[n] = ""
		}
	}
	if numCards < (packSize - 7) {
		prevNum := numCards + 8
		previousContents[numCards] = packContents[prevNum]
	}
	// Do some computations to figure out the optimal picks for plat, gold and filling out our collection
	contentsInfo := ""
	for _, u := range cards {
		card := u.(map[string]interface{})
		uuid := getCardUUIDFromJSON(card)
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
		// The first time we have a blank comma at the end, but we remove that later
		packContents[numCards] = fmt.Sprintf("'%v', %v", c.name, packContents[numCards])
		contentsInfo = fmt.Sprintf("'[%v %2d - %3dp/%3dg] %v'\n\t%v", c.rarity, c.qty, c.plat, c.gold, c.name, contentsInfo)

		// If we have (packSize - 7) or more cards in pack, save what we've got so we've got so we can determine what others picked
		if numCards < (packSize - 7) {
			prevCard := fmt.Sprintf("'%v', ", c.name)
			previousContents[numCards] = strings.Replace(previousContents[numCards], prevCard, "", 1)
		}
	}
	// Curious why I'm doing this. Pretty sure it's to remove the leading ", "from the string
	packContents[numCards] = strings.Replace(packContents[numCards], ", ", "", 1)
	contentsInfo = strings.Replace(contentsInfo, ", ", "", 1)
	fmt.Printf("== Pack [%v] Contents:\n\t%v", numCards, contentsInfo)
	if numCards < (packSize - 7) {
		fmt.Printf("-- MISSING CARDS: %v\n", previousContents[numCards])
	}
	mostGold := getCardInfo(worthMostGold)
	mostPlat := getCardInfo(worthMostPlat)
	haveLeast := getCardInfo(haveLeastOf)
	fmt.Println("** Computed best picks from pack:")
	fmt.Printf("\tWorth most plat: %v\n", mostPlat)
	fmt.Printf("\tWorth most gold: %v\n", mostGold)
	fmt.Printf("\tHave least of: %v\n", haveLeast)
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
	// First thing we do is stop the timer
	collectionCacheTimer.Stop()
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
	}
	f.Sync()

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
		fmt.Printf("Writing CSV card data to file '%v'.", csvFile)
		// Defer our close
		defer f.Close()

		// Get a sorted list of cards
		nmk := listCardsSortedByName()
		// Then use that array to print out card info in Alphabetical order
		for _, name := range nmk {
			uuid := ntum[name]
			c := cardCollection[uuid]
			if Config["detailed_card_info"] == "true" {
				fmt.Printf("CSV caching '%v [%v]' with qty %v\n", c.name, c.uuid, c.qty)
			}
			if c.qty > 0 {
				line := fmt.Sprintf("\"%v\",%v\n", c.name, c.qty)
				f.WriteString(line)
			}
		}
	}
	fmt.Printf("\n")
	if Config["show_collection_value"] == "true" {
		printCollectionValue()
	}
}

func logAPICall(line string) {
	// Open file. If it exists right now, remove that before creating a new one
	logAPIFile := Config["api_log_file"]
	f, err := os.OpenFile(logAPIFile, os.O_RDWR|os.O_APPEND, 0660)
	if err != nil {
		fmt.Printf("Could not append to file %v for writing: %v\n", logAPIFile, err)
		return
	}
	// fmt.Printf("Writing API call to file '%v'.\n", logAPIFile)
	// Defer our close
	defer f.Close()

	// Write out the line
	f.WriteString(line + "\n")
	f.Sync()
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
	for _, u := range added {
		card := u.(map[string]interface{})
		uuid := getCardUUIDFromJSON(card)
		if uuid == "00000000-0000-0000-0000-000000000000" {
			continue
		}
		name := ""
		if _, ok := cardCollection[uuid]; ok {
			// Card exists. Do a straight update
			incrementCardCount(uuid)
		} else {
			// If it doesn't exist, create a new card with appropriate values and add it to the map
			name = getCardNameFromJSON(card)
			// Make up a bogus rarity
			rarity := "?"
			// See if we've already got another version of this card in our list
			if _, ok := ntum[name]; ok {
				// We have a card with a different UUID and the same name. That means this card
				// is likely an AA.  We'll double check and add the new card as appropriate.
				if strings.Contains(name[len(name)-3:], " AA") {
					name = name[len(name)-3:]
				} else {

					newName := name + " AA"
					rarity = "E"
					fmt.Printf("While adding missing card, changing name of '%v' to '%v' because its rarity is '%v'\n", name, newName, rarity)
					name = newName
				}
			}
			// If we don't have pricing, set plat and gold to 1. Also, set qty to 1 so we don't have
			// to do an 'incrementCardCount(uuid) afterward.
			c := Card{name: name, uuid: uuid, plat: 1, gold: 1, rarity: rarity, qty: 1}
			cardCollection[uuid] = c
		}
		if Config["debug_collection_update"] == "true" {
			fmt.Printf("INFO: [collectionEvent] Adding [%s] %s for count of %d in collection\n", uuid, card, getCardCount(uuid))
		}
	}
	for _, u := range removed {
		card := u.(map[string]interface{})
		uuid := getCardUUIDFromJSON(card)
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
	resetGame()
}

// Message: {"Players":[],"User":"InGameName","Message":"GameStarted"}
func gameStartedEvent() {
	GameStartTime = time.Now()
	fmt.Printf("Game started at %v\n", GameStartTime.Format(time.UnixDate))
	resetGame()
}

// Set up game in progress
func resetGame() {
	pnums := map[string]int{"p1": 1, "p2": 2}
	player1 := Player{name: "p1", id: 1, champion: gameCard{name: "Unknown"}}
	player2 := Player{name: "p2", id: 2, champion: gameCard{name: "Unknown"}}
	currentGame = Game{p1: player1, p2: player2, pnums: pnums}
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

// Do something meaningful with the playerUpdated Event
func playerUpdatedEvent(f map[string]interface{}) {
	var p Player
	var pptr *Player
	var msg string
	// If this is the first time through, do a resetGame
	if currentGame.p1.id == 0 {
		resetGame()
	}
	// See if this player is config'd and figure out if it's p1 or p2
	pptr = checkPlayerConfigured(f["Id"])
	// Go ahead and make the updates
	// p, c = updatePlayer(pptr, res, thresholds)
	p, msg = updatePlayer(pptr, f)
	if msg != "" {
		fmt.Printf("%v", msg)
		if Config["debug_player_update"] == "true" {
			fmt.Printf("PlayerUpdate for %v\n", p)
		}
	}
}

// See if this player ID has been allocated for current game yet. If not, add it to the next
// empty slot in the current game
func checkPlayerConfigured(fID interface{}) *Player {
	id := floatToInt(fID)
	// If this is 1, the first player hasn't been initialized yet.
	if currentGame.pnums["p1"] == id {
		return &currentGame.p1
	} else if currentGame.pnums["p2"] == id {
		return &currentGame.p2
	} else if currentGame.pnums["p1"] == 1 {
		if Config["debug_player_configured"] == "true" {
			fmt.Printf("Overwriting p1 id with %v (prev id was %v)\n", id, currentGame.pnums["p1"])
		}
		currentGame.pnums["p1"] = id
		currentGame.p1.id = id
		return &currentGame.p1
	} else
	// same for this for player 2
	if currentGame.pnums["p2"] == 2 {
		if Config["debug_player_configured"] == "true" {
			fmt.Printf("Overwriting p2 id with %v (prev id was %v)\n", id, currentGame.pnums["p2"])
		}
		currentGame.pnums["p2"] = id
		currentGame.p2.id = id
		return &currentGame.p2
	} else {
		// Print an error, then return p1 from the current game
		if Config["debug_player_configured"] == "true" {
			fmt.Printf("Got somewhere I should not have in checkPlayerConfigured. pnums[p1]: %v, pnums[p2]: %v, id: %v\nResetting game and trying again.", currentGame.pnums["p1"], currentGame.pnums["p2"], id)
		}
		resetGame()
		currentGame.pnums["p1"] = id
		currentGame.p1.id = id
		return &currentGame.p1
	}
}

func valuesDiffer(a int, b int) bool {
	if a != b {
		fmt.Printf("Values differ: %d vs %d\n", a, b)
		return true
	}
	return false
}

func updatePlayer(p *Player, f map[string]interface{}) (Player, string) {
	// The thing we pass back to say whether or not this changed.
	msg := ""
	// Get thresholds hash
	t := f["Thresholds"].(map[string]interface{})
	// Turn resources into an int
	r := floatToInt(f["Resources"])
	// If the player's name has been set, use that for printing things out
	name := ""
	if p.name != "" {
		name = p.name
	} else {
		name = strconv.Itoa(p.id)
	}

	msg = compareIntsAndMaybeChange(&p.resources, r, "Number of resources", msg)
	msg = compareIntsAndMaybeChange(&p.blood, floatToInt(t["Blood"]), "Blood threshold", msg)
	msg = compareIntsAndMaybeChange(&p.diamond, floatToInt(t["Diamond"]), "Diamond threshold", msg)
	msg = compareIntsAndMaybeChange(&p.ruby, floatToInt(t["Ruby"]), "Ruby threshold", msg)
	msg = compareIntsAndMaybeChange(&p.sapphire, floatToInt(t["Sapphire"]), "Sapphire threshold", msg)
	msg = compareIntsAndMaybeChange(&p.wild, floatToInt(t["Wild"]), "Wild threshold", msg)

	if msg != "" {
		msg = fmt.Sprintf("The following changed for %v\n%v", name, msg)
	}

	return *p, msg
}

func compareIntsAndMaybeChange(a *int, b int, name string, m string) string {
	if *a != b {
		m = fmt.Sprintf("%v\t%v => %v : %v changed\n", m, *a, b, name)
		*a = b
	}
	return m
}

func floatToInt(f interface{}) int {
	stringI := fmt.Sprintf("%v", f)
	i, _ := strconv.Atoi(stringI)
	return i
}

func saveDeckEvent(f map[string]interface{}) {
	fmt.Println("In function of saveDeckEvent")
}

func dumpRequest(rw http.ResponseWriter, req *http.Request) {
	fmt.Println("Request to print collection recieved.")
	printCollection()
}

func valueRequest(rw http.ResponseWriter, req *http.Request) {
	fmt.Println("Request to print value of collection recieved.")
	printCollectionValue()
}

func printCollectionValue() {
	// Zero out the gold and plat values
	collectionGoldValue = 0
	collectionPlatValue = 0
	for _, v := range cardCollection {
		// Skip cards that we don't have any of
		if v.qty == 0 {
			continue
		}
		// Add the appropriate values to the appropriate variables
		collectionPlatValue = collectionPlatValue + (v.plat * v.qty)
		collectionGoldValue = collectionGoldValue + (v.gold * v.qty)
	}
	fmt.Printf("Your collection is currently valued at %v plat and %v gold\n", collectionPlatValue, collectionGoldValue)
}

func incoming(rw http.ResponseWriter, req *http.Request) {
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		panic("AIEEE: Could not readAll for req.Body")
	}
	var f map[string]interface{}
	// fmt.Printf("Contents of body:\n\t%v\n", string(body))
	err = json.Unmarshal(body, &f)
	if err != nil {
		fmt.Printf("Could not unmarshall the following body:\n\t%v\n", string(body))
		return
		// panic("AIEEE: Could not Unmarshall the body")
	}
	//	fmt.Println("DEBUG: Unmarshall successful")
	//	fmt.Println("DEBUG: Message is", f["Message"])
	skipDupes := true
	msg := f["Message"]
	if msg == "Collection" {
		if f["Action"] == "Update" {
			skipDupes = false
		}
	}
	// Check to see if the last API message is the same as this one (as long as we're not in a Collection Update)
	if lastAPIMessage == string(body) && skipDupes {
		// fmt.Println("INFO: Duplicate API Message. Discarding.")
		return
	}
	lastAPIMessage = string(body)
	// If we want to log API calls, make use of the lastAPIMessage we just set and log it here
	if Config["log_api_calls"] == "true" {
		logAPICall(lastAPIMessage)
	}
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
		playerUpdatedEvent(f)
	default:
		fmt.Printf("Don't know how to handle message '%v'\n", msg)
	}
}

func loadDefaults() map[string]string {
	retMap := make(map[string]string)
	retMap["price_url"] = "http://doc-x.net/hex/all_prices_json.txt"
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
	//  Retrieve from http://doc-x.net/hex/all_prices_json.txt

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
		// s := string(body)
		var f map[string]interface{}
		err = json.Unmarshal(body, &f)
		if err != nil {
			panic("Could not Unmarshal price body")
		}
		// fmt.Println("Unmarshalled price Body")
		cards := f["cards"].([]interface{})
		// fmt.Println("Assigned cards from 'f' and made cj variable")
		// Make some variables so we re-use them instead of re-creating them each run through
		var c = make(map[string]interface{})
		var name string
		var rarity string
		var fullRarity string
		var uuid string
		var p = make(map[string]interface{})
		var plat int
		var g = make(map[string]interface{})
		var gold int

		// Reduce the spamminess of loading collection info
		loadingCacheOrPriceData = true

		for _, card := range cards[:] {
			c = card.(map[string]interface{})
			// Verify this actually has a thing name. If it doesn't, go to the next thing
			if c["name"] == nil {
				continue
			}
			// Assign our variables from the interface derived from our JSON blob
			name = c["name"].(string)
			fullRarity = c["rarity"].(string)
			if len(fullRarity) > 0 {
				rarity = fullRarity[:1]
			} else {
				rarity = "?"
			}
			uuid = c["uuid"].(string)
			p = c["PLATINUM"].(map[string]interface{})
			plat = int(p["avg"].(float64))
			g = c["GOLD"].(map[string]interface{})
			gold = int(g["avg"].(float64))
			// fmt.Printf("Working on '%v'\nName is '%v', rarity is %v and uuid is %v and avg plat of %v and avg gold of %v\n", card, name, rarity, uuid, plat, gold)
			// If we've already got a card with that UUID in the cardCollection, update the info
			if _, ok := cardCollection[uuid]; ok {
				// We can't update directly, so we create a new card, modify it's values, then reassign it back to
				// to cardCollection
				c := cardCollection[uuid]
				c.name = name
				c.uuid = uuid
				c.plat = plat
				c.gold = gold
				c.rarity = rarity
				cardCollection[uuid] = c
			} else {
				// If it doesn't exist, create a new card with appropriate values and add it to the map
				c := Card{name: name, uuid: uuid, plat: plat, gold: gold, rarity: rarity}
				cardCollection[uuid] = c
				// And update our name to uuid map
				ntum[name] = uuid
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
		packGoldCost = draftPack.gold
		goldPlatRatio = int(draftPack.gold / draftPack.plat)
	}

	// If this has been set, go through all cards, find any with 0 for gold or plat value, then compute it from the other value.
	// If both are 0, set value to 1g and 1p.
	if goldPlatRatio > 0 {
		for k, v := range cardCollection {
			if v.plat == 0 && v.gold == 0 {
				c := cardCollection[k]
				c.plat = 1
				c.gold = 1
				cardCollection[k] = c
			}
			if v.plat == 0 {
				c := cardCollection[k]
				c.plat = int(c.gold / goldPlatRatio)
				// In cases where this is actually zero after the comparison, go ahead and make it a minimum of 1
				if c.plat == 0 {
					c.plat = 1
				}
				cardCollection[k] = c
				continue
			}
			if v.gold == 0 {
				c := cardCollection[k]
				c.gold = c.plat * goldPlatRatio
				cardCollection[k] = c
				continue
			}
		}

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

// Make sure we don't fill up our disk by logging API data
func truncateAPILogFile() {
	if Config["log_api_calls"] == "true" {
		logAPIFile := Config["api_log_file"]
		if _, err := os.Stat(logAPIFile); err != nil {
			fmt.Printf("Creating API log file '%s' as it does not currently exist\n", Config["api_log_file"])
			f, err := os.Create(logAPIFile)
			if err != nil {
				fmt.Printf("Could not create file %v for writing: %v\n", logAPIFile, err)
				return
			}
			f.Close()
		}
		fmt.Printf("Truncating API log file '%s' so we don't fill up the disk\n", Config["api_log_file"])
		os.Truncate(Config["api_log_file"], 0)
	}
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
	// Run this to truncate API log file if we are logging
	truncateAPILogFile()
	fmt.Println("Beginning to listen for API events")
	// Register http handlers before starting the server
	http.HandleFunc("/", incoming)
	http.HandleFunc("/dump", dumpRequest)
	http.HandleFunc("/value", valueRequest)
	// Now that we've registered what we want, start it up
	log.Fatal(http.ListenAndServe(":5000", nil))
}

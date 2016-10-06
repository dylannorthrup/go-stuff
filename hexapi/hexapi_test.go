// Test cases for hexapi stuff

package main

import "testing"

func TestValuesDiffer(t *testing.T) {
	for _, c := range []struct {
		a, b int
		want bool
	}{
		{1, 1, false},
		{1, 2, true},
		{0, -0, false},
		{-1, 1, true},
	} {
		// a := 1
		// b := 1
		Config = make(map[string]string)
		Config["running_test"] = "true"

		got := valuesDiffer(c.a, c.b)
		if got != c.want {
			t.Errorf("valuesDiffer(%q, %q) == %v but we expected %v", c.a, c.b, got, c.want)
		}
	}
}

func TestFloatToInt(t *testing.T) {
	for _, c := range []struct {
		a    float64
		want int
	}{
		{1.0, 1},
		{2.0, 2},
		{-0.0, 0},
		{-1.0, -1},
	} {
		got := floatToInt(c.a)
		if got != c.want {
			t.Errorf("floatToInt(%v) == %v but we expected %v", c.a, got, c.want)
		}
	}
}

func TestObjRawChangeCardCount(t *testing.T) {
	uuid := "ff9f4b37-6b97-4cc6-bbde-87974f1bb678"
	for _, nums := range []struct {
		a int
	}{
		{1}, {2}, {0}, {100},
	} {
		c := Card{name: "Test Card", uuid: uuid, plat: 1, gold: 1, rarity: "Epic", qty: 0, nature: "Smelly"}
		c.objRawChangeCardCount(nums.a)
		q := c.qty
		if q != nums.a {
			t.Errorf("c.rawChangeCardCount(%v) == %v but we expected %v", nums.a, c.qty, nums.a)
		}

	}
}

func TestRawChangeCardCount(t *testing.T) {
	uuid := "ff9f4b37-6b97-4cc6-bbde-87974f1bb678"
	// Give us some more verbose debugging output
	// Config = make(map[string]string)
	// Config["detailed_card_info"] = "true"
	for _, nums := range []struct {
		a int
	}{
		{1}, {2}, {0}, {100},
	} {
		c := Card{name: "Test Card", uuid: uuid, plat: 1, gold: 1, rarity: "Epic", qty: 0, nature: "Smelly"}
		cardCollection[uuid] = c
		rawChangeCardCount(uuid, nums.a)
		nc := cardCollection[uuid]
		if nc.qty != nums.a {
			t.Errorf("c.rawChangeCardCount(%v) == %v but we expected %v\n\tCARDINFO: %v", nums.a, nc.qty, nums.a, nc.InfoWithWheelInfo(0))
		}

	}
}

func TestRawChangeEACardCount(t *testing.T) {
	uuid := "ff9f4b37-6b97-4cc6-bbde-87974f1bb678"
	for _, nums := range []struct {
		a int
	}{
		{1}, {2}, {0}, {100},
	} {
		c := Card{name: "Test Card", uuid: uuid, plat: 1, gold: 1, rarity: "Epic", qty: 0, eaqty: 0, nature: "Smelly"}
		cardCollection[uuid] = c
		rawChangeEACardCount(uuid, nums.a)
		nc := cardCollection[uuid]
		if nc.eaqty != nums.a {
			t.Errorf("c.rawChangeCardCount(%v) == %v but we expected %v\nCARDINFO: %v\n", nums.a, nc.qty, nums.a, nc.InfoWithWheelInfo(0))
		}
	}
}

func TestEARawChangeCardCount(t *testing.T) {
	uuid := "ff9f4b37-6b97-4cc6-bbde-87974f1bb678"
	// Give us some more verbose debugging output
	// Config = make(map[string]string)
	// Config["detailed_card_info"] = "true"
	for _, nums := range []struct {
		a int
	}{
		{1}, {2}, {0}, {100},
	} {
		c := Card{name: "Test Card", uuid: uuid, plat: 1, gold: 1, rarity: "Epic", qty: 0, nature: "Smelly"}
		cardCollection[uuid] = c
		rawChangeCardCount(uuid, nums.a)
		nc := cardCollection[uuid]
		if nc.qty != nums.a {
			t.Errorf("c.rawChangeCardCount(%v) == %v but we expected %v\n\tCARDINFO: %v", nums.a, nc.qty, nums.a, nc.InfoWithWheelInfo(0))
		}

	}
}

func TestNatureTranslation(t *testing.T) {
	for _, a := range []struct {
		given string
		want  string
	}{
		{"Equipment", "Inventory"},
		{"Card", "Card"},
		{"FooBar", "Unknown"},
		{"", "Unknown"},
	} {
		got := translateCardNature(a.given)
		if got != a.want {
			t.Errorf("translateCardNature(%v) == %v but we expected %v\n", a.given, got, a.want)
		}
	}
}

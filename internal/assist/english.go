package assist

import "strings"

// singularOf is EnglishNoun.singularOf of the Java plug-in (James A. Mason, 2002): the
// singular of a presumed English noun, in lower case.
func singularOf(word string) string {
	word = strings.ToLower(word)
	if isSingular(word) {
		return word
	}
	if s, ok := irregularPlurals[word]; ok {
		return s
	}
	r := []rune(word)
	n := len(r)
	if n <= 1 || r[n-1] != 's' {
		return word
	}
	switch {
	case r[n-2] == '\'':
		return string(r[:n-2]) // -'s
	case word == "gas":
		return word
	case r[n-2] != 'e' || n <= 3:
		return string(r[:n-1]) // -s
	}
	// -es, at least 4 letters
	switch {
	case r[n-3] == 'i':
		return string(r[:n-3]) + "y" // -ies
	case r[n-3] == 'x':
		return string(r[:n-2]) // -xes
	case n <= 4:
		return string(r[:n-1]) // uses -> use
	case r[n-3] == 'h' && (r[n-4] == 'c' || r[n-4] == 's'):
		return string(r[:n-2]) // -ches, -shes
	case r[n-3] == 's' && r[n-4] == 's':
		return string(r[:n-2]) // -sses
	}
	return string(r[:n-1]) // keep the final e
}

func isSingular(word string) bool {
	if irregularSingulars[word] {
		return true
	}
	if _, ok := irregularPlurals[word]; ok {
		return false
	}
	n := len(word)
	if n == 0 {
		return false
	}
	if word[n-1] != 's' {
		return true
	}
	return n >= 2 && word[n-2] == 's'
}

var irregularSingulars = map[string]bool{
	"ache": true, "alumna": true, "alumnus": true, "axis": true, "bison": true, "bus": true,
	"calf": true, "caribou": true, "child": true, "datum": true, "deer": true, "die": true,
	"elf": true, "elk": true, "fish": true, "foot": true, "gentleman": true, "gentlewoman": true,
	"go": true, "goose": true, "grouse": true, "half": true, "hoof": true, "knife": true,
	"leaf": true, "life": true, "louse": true, "man": true, "money": true, "moose": true,
	"mouse": true, "octopus": true, "ox": true, "plus": true, "quail": true, "reindeer": true,
	"scarf": true, "self": true, "sheaf": true, "sheep": true, "shelf": true, "squid": true,
	"thief": true, "tooth": true, "wharf": true, "wife": true, "wolf": true, "woman": true,
}

// irregularPlurals: where Java keeps several singulars, the first one.
var irregularPlurals = map[string]string{
	"aches": "ache", "alumnae": "alumna", "alumni": "alumnus", "axes": "axe", "bison": "bison",
	"buses": "bus", "busses": "bus", "brethren": "brother", "caches": "cache", "calves": "calf",
	"cargoes": "cargo", "caribou": "caribou", "children": "child", "data": "datum", "deer": "deer",
	"dice": "die", "dies": "die", "dominoes": "domino", "echoes": "echo", "elves": "elf", "elk": "elk",
	"embargoes": "embargo", "fish": "fish", "feet": "foot", "gentlemen": "gentleman",
	"gentlewomen": "gentlewoman", "geese": "goose", "goes": "go", "grottoes": "grotto",
	"grouse": "grouse", "halves": "half", "hooves": "hoof", "knives": "knife", "leaves": "leaf",
	"lives": "life", "lice": "louse", "men": "man", "monies": "money", "moose": "moose",
	"mottoes": "motto", "mice": "mouse", "octopi": "octopus", "octopodes": "octopus",
	"octopuses": "octopus", "oxen": "ox", "pies": "pie", "pluses": "plus", "posses": "posse",
	"potatoes": "potato", "quail": "quail", "reindeer": "reindeer", "scarves": "scarf",
	"sheaves": "sheaf", "sheep": "sheep", "shelves": "shelf", "squid": "squid", "teeth": "tooth",
	"thieves": "thief", "ties": "tie", "tomatoes": "tomato", "wharves": "wharf", "wives": "wife",
	"wolves": "wolf", "women": "woman",
}

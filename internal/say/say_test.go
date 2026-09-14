package say

import "testing"

// Russian has three forms of a number, and which one is not a matter of one
// and many: 21 takes the first, 22 the second, 25 the third.
func TestCountsTheRussianWay(t *testing.T) {
	Speak(Russian)
	defer Speak(English)

	for number, wanted := range map[int]string{
		1: "1 раз", 2: "2 раза", 5: "5 раз",
		11: "11 раз", 21: "21 раз", 22: "22 раза", 25: "25 раз",
		101: "101 раз", 112: "112 раз",
	} {
		if got := Count(number, "times"); got != wanted {
			t.Errorf("%d: got %q, wanted %q", number, got, wanted)
		}
	}
}

func TestCountsTheEnglishWay(t *testing.T) {
	Speak(English)

	for number, wanted := range map[int]string{1: "1 time", 2: "2 times", 21: "21 times"} {
		if got := Count(number, "times"); got != wanted {
			t.Errorf("%d: got %q", number, got)
		}
	}
}

// French keeps the singular for nought as well as for one.
func TestCountsTheFrenchWay(t *testing.T) {
	Speak(French)
	defer Speak(English)

	if got := Count(1, "lead.kept"); got != ", 1 lecture gardée." {
		t.Errorf("got %q", got)
	}

	if got := Count(3, "lead.kept"); got != ", 3 lectures gardées." {
		t.Errorf("got %q", got)
	}
}

// A language nobody wrote down leaves the one that was being spoken.
func TestKeepsToWhatItKnows(t *testing.T) {
	Speak(English)
	Speak("kl")

	if Spoken() != English {
		t.Errorf("wandered off to %q", Spoken())
	}
}

// Every line said in one language should be said in all of them.
func TestSaysEverythingInEveryLanguage(t *testing.T) {
	for key := range words[English] {
		for _, tongue := range Languages() {
			if _, ok := words[tongue][key]; !ok {
				t.Errorf("%q is not said in %s", key, tongue)
			}
		}
	}
}

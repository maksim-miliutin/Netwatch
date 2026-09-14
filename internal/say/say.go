// Package say holds what the page says, in the languages it says it in.
package say

import (
	"strings"
	"sync"
)

const (
	English = "en"
	Russian = "ru"
	French  = "fr"
)

func Languages() []string {
	return []string{English, Russian, French}
}

func Named(tongue string) string {
	switch tongue {
	case Russian:
		return "Русский"
	case French:
		return "Français"
	}

	return "English"
}

var (
	minding sync.RWMutex
	tongue  = English
)

func Speak(which string) {
	minding.Lock()
	defer minding.Unlock()

	for _, known := range Languages() {
		if which == known {
			tongue = which

			return
		}
	}
}

func Spoken() string {
	minding.RLock()
	defer minding.RUnlock()

	return tongue
}

// In returns what is said for a key, and the key itself when nothing is: a
// missing line should be visible rather than blank.
func In(key string) string {
	said, ok := words[Spoken()][key]
	if !ok {
		if said, ok = words[English][key]; !ok {
			return key
		}
	}

	return said
}

// Count picks the form a number takes. English and French have two; Russian
// has three, and which one is not a matter of one and many.
func Count(number int, key string) string {
	forms := strings.Split(In(key), "|")

	return strings.Replace(forms[form(number, len(forms))], "#", digits(number), 1)
}

func form(number, forms int) int {
	if forms < 3 {
		if Spoken() == French && number < 2 {
			return 0
		}

		if number == 1 {
			return 0
		}

		return forms - 1
	}

	last, hundred := number%10, number%100

	if last == 1 && hundred != 11 {
		return 0
	}

	if last >= 2 && last <= 4 && (hundred < 12 || hundred > 14) {
		return 1
	}

	return 2
}

func digits(number int) string {
	if number == 0 {
		return "0"
	}

	var written []byte

	for number > 0 {
		written = append([]byte{byte('0' + number%10)}, written...)
		number /= 10
	}

	return string(written)
}

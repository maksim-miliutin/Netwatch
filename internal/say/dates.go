package say

import "time"

var months = map[string][]string{
	English: {"January", "February", "March", "April", "May", "June", "July",
		"August", "September", "October", "November", "December"},
	Russian: {"января", "февраля", "марта", "апреля", "мая", "июня", "июля",
		"августа", "сентября", "октября", "ноября", "декабря"},
	French: {"janvier", "février", "mars", "avril", "mai", "juin", "juillet",
		"août", "septembre", "octobre", "novembre", "décembre"},
}

var weekdays = map[string][]string{
	English: {"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday",
		"Saturday"},
	Russian: {"воскресенье", "понедельник", "вторник", "среда", "четверг",
		"пятница", "суббота"},
	French: {"dimanche", "lundi", "mardi", "mercredi", "jeudi", "vendredi",
		"samedi"},
}

// Go knows one language for dates, and it is not one of these.
func Day(when time.Time) string {
	which := Spoken()

	weekday := weekdays[which][int(when.Weekday())]
	month := months[which][int(when.Month())-1]

	if which == English {
		return weekday + ", " + digits(when.Day()) + " " + month
	}

	return weekday + ", " + digits(when.Day()) + " " + month
}

func Date(when time.Time) string {
	return digits(when.Day()) + " " + months[Spoken()][int(when.Month())-1]
}

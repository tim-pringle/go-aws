package transcribe2srt

import (
	"fmt"
	"math"
	"strconv"
)

// Getsrttime - Generates an SRT format time string
func Getsrttime(numerator float64) (timestring string) {

	var h = 3600
	var m = 60
	var s = 1

	integer, frac := math.Modf(numerator)
	integerpart := int(integer)

	hours := integerpart / h
	remainder := integerpart % h

	minutes := remainder / m
	remainder = remainder % m

	seconds := remainder / s
	stringfrac := strconv.FormatFloat(frac, 'f', 3, 64)
	runes := []rune(stringfrac)
	safeSubstring := string(runes[1:len(stringfrac)])

	timestring = fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
	timestring += safeSubstring
	return
}

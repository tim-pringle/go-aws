package convert

import (
	"fmt"
	"strconv"
	"time"
)

//GUID - generates a unique identifier
func GUID() (guid string) {
	ad, err := time.Parse("02-01-2006", "01-01-1970")

	if err != nil {
	}

	timesince := time.Since(ad).Nanoseconds()
	strsince := strconv.FormatInt(timesince, 10)
	guid = fmt.Sprintf("0" + strsince[0:4] + "-" + strsince[4:9] + "-" + strsince[9:14] + "-" + strsince[14:19])
	return
}

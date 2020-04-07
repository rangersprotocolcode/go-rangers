package utility

import (
	"fmt"
	"testing"
	"time"
)

func TestQuery(t *testing.T) {
	offset, err := NTPOffset()
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println(offset)
	time := time.Now().Add(offset)
	fmt.Println(time)

}

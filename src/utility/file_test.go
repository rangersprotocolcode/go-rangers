package utility

import (
	"testing"
	"fmt"
)

func TestMd5SumFolder(t *testing.T) {
	result, err := checkFolderDetail("/Users/daijia/go/src/x/src/statemachine", 10)
	if err != nil {
		t.Fatal(err)
	}

	for key, value := range result {
		fmt.Printf("%s, %v\n", key, value)
	}

}

func TestCheckFolder(t *testing.T) {
	result, detail := CheckFolder("./")
	fmt.Printf("%v\n", result)

	fmt.Println("details:")
	for i, item := range detail {
		fmt.Printf("%d %v \n", i, item)
	}

}

func TestZip(t *testing.T) {
	err := Zip("/Users/daijia/go/src/x/src/statemachine/logs", "1111.zip")
	if err != nil {
		panic(err)
	}

}

func TestUnzip(t *testing.T) {
	Unzip("1111.zip", "")
}

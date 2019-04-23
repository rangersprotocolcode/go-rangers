package core

import (
	"testing"
	"encoding/json"
	"fmt"
	"log"
)

func TestJson(t *testing.T){
	w := WithdrawInfo{
		Address: "fss",
		GameId: "fsfsaf",
	}

	b, err := json.Marshal(w)
	if err != nil {
		fmt.Printf("Json marshal withdrawInfo err:%s", err.Error())
		return
	}
	fmt.Printf("%v\n",b)
	fmt.Println(string(b))
}

type Account struct {
	Email string
	password string
	Money float64
}

func TestJson1(t *testing.T){
	account := Account{
		Email: "rsj217@gmail.com",
		password: "123456",
		Money: 100.5,
	}

	rs, err := json.Marshal(account)
	if err != nil{
		log.Fatalln(err)
	}

	fmt.Println(rs)
	fmt.Println(string(rs))
}
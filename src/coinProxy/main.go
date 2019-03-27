package main

import (
"net/http"
"strings"
"fmt"
"io/ioutil"
)

func main() {
	resp, err := http.Post("http://www.maimaiche.com/loginRegister/login.do",
		"application/x-www-form-urlencoded",
		strings.NewReader("mobile=xxxxxxxxxx&isRemberPwd=1"))
	if err != nil {
		fmt.Println(err)
		return
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(string(body))
}

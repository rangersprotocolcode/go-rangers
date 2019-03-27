package storestate

import (
	"fmt"
	"database/sql"
	_"github.com/go-sql-driver/mysql"
	"math/big"
)

type StoreState struct{
	db	*sql.DB
}

func (self *StoreState)Init() error{
	var err error
	self.db, err = sql.Open("mysql",
		"root:123456@tcp(localhost:3306)/coinproxy?charset=utf8")
	if err != nil {
		panic(err)
		return err
	}

	return nil
}

func (self *StoreState)Deinit() error {
	if self.db !=nil{
		self.db.Close()
	}
	return nil
}

func (self *StoreState)AddGame(cointype string,gamename string,contractaddress string) error {
	stmt1, _ := self.db.Prepare(`SELECT * From contracts where contractaddress=?`)
	defer stmt1.Close()

	ret1, err := stmt1.Exec(contractaddress)
	if err != nil{
		fmt.Printf("AddGame select err: %v\n", err)
		return err
	}
	count,err := ret1.RowsAffected()
	if err != nil {
		fmt.Printf("AddGame RowsAffected err: %v\n", err)
		return err
	}
	if count>0 {
		fmt.Printf("AddGame has alreadly exist")
		return nil
	}

	stmt, _ := self.db.Prepare(`INSERT INTO contracts (cointype,gamename ,contractaddress,time) VALUES (?,?,?,now())`)
	defer stmt.Close()

	ret, err := stmt.Exec(cointype, gamename,contractaddress)
	if err != nil {
		fmt.Printf("AddGame error: %v\n", err)
		return err
	}
	if _, err := ret.LastInsertId(); nil != err {
		return err
		//fmt.Println("LastInsertId:", LastInsertId)
	}
	if _, err := ret.RowsAffected(); nil != err {
		//fmt.Println("RowsAffected:", RowsAffected)
		return err
	}
	return nil
}

func (self *StoreState)DelGame(cointype string,contractaddress string) error {
	stmt, _ := self.db.Prepare(`DELETE FROM contracts WHERE cointype=? AND contractaddress=?`)
	defer stmt.Close()

	ret, err := stmt.Exec(cointype, contractaddress)
	if err != nil {
		fmt.Printf("DelGame error: %v\n", err)
		return err
	}
	if _, err := ret.LastInsertId(); nil != err {
		return err
		//fmt.Println("LastInsertId:", LastInsertId)
	}
	if _, err := ret.RowsAffected(); nil != err {
		//fmt.Println("RowsAffected:", RowsAffected)
		return err
	}
	return nil
}

func (self *StoreState)GetAllGames() (map[string]string,error) {
	addresss:=map[string]string{}
	rows,err := self.db.Query(`SELECT cointype,contractaddress FROM contracts`)
	if err != nil {
		return nil,err
	}
	defer rows.Close()

	for rows.Next() {
		var col1,col2 sql.NullString
		err = rows.Scan(&col1,&col2)
		if err != nil {
			return nil,err
		}
		addresss[col2.String] = col1.String
	}
	if err = rows.Err(); err != nil {
		return nil,err
	}
	return addresss,nil


	//stmt1, _ := self.db.Prepare(`SELECT contractaddress FROM contracts WHERE cointype=eth`)
	//defer stmt1.Close()

	//ret1, err := stmt1.Exec()
	//if err != nil{
	//	fmt.Printf("AddGame select err: %v\n", err)
	//	return nil,err
	//}
	//count,err := ret1.RowsAffected()
	//if err != nil {
	//	fmt.Printf("AddGame RowsAffected err: %v\n", err)
	//	return nil,err
	//}
	//if count>0 {
	//	fmt.Printf("AddGame has alreadly exist:%v")
	//	return nil
	//}
	//return nil
}

func (self *StoreState)AddIncoming(contractaddress string,txhash string,data string,topics string,blocknumber big.Int) error {
	stmt1, err := self.db.Prepare(`SELECT * FROM callinfo WHERE transferhash=?`)
	if err != nil {
		return err
	}
	defer stmt1.Close()

	ret1, err := stmt1.Exec(txhash)
	if err != nil{
		fmt.Printf("AddIncoming select err: %v\n", err)
		return err
	}
	count,err := ret1.RowsAffected()
	if err != nil {
		fmt.Printf("AddIncoming RowsAffected err: %v\n", err)
		return err
	}
	if count>0 {
		fmt.Printf("this is not AddIncoming")
		return nil
	}

	stmt, _ := self.db.Prepare(`INSERT INTO incoming (contractaddress, txhash,data,topics,blocknumber,addtime) VALUES (?,?,?,?,?,now())`)
	defer stmt.Close()

	ret, err := stmt.Exec(contractaddress, txhash,data,topics,blocknumber.Int64())
	if err != nil {
		fmt.Printf("AddIncoming error: %v\n", err)
		return err
	}

	_, err = ret.LastInsertId()
	if nil != err {
		return err
		//fmt.Println("LastInsertId:", LastInsertId)
	}

	if _, err := ret.RowsAffected(); nil != err {
		//fmt.Println("RowsAffected:", RowsAffected)
		return err
	}
	return nil
}

func (self *StoreState)AddInfo(cointype string,fromaddr string,toaddr string,inputdata []byte) (id int64,erro error){
	stmt, _ := self.db.Prepare(`INSERT INTO callinfo (cointype, fromaddr,toaddr,inputdata,status,addtime) VALUES (?,?,?,?,?,now())`)
	defer stmt.Close()

	ret, err := stmt.Exec(cointype, fromaddr,toaddr,inputdata,"init")
	if err != nil {
		fmt.Printf("AddInfo error: %v\n", err)
		return 0,err
	}

	id, err = ret.LastInsertId()
	if nil != err {
		return 0,err
		//fmt.Println("LastInsertId:", LastInsertId)
	}

	if _, err := ret.RowsAffected(); nil != err {
		//fmt.Println("RowsAffected:", RowsAffected)
		return 0,err
	}
	return id,nil
}

func (self *StoreState)UpdatePending(id int64,gasprice big.Int,transferhash string) error{
	stmt, _ := self.db.Prepare(`UPDATE callinfo SET status=?,gasprice=?,transferhash=?,pendingtime=now() WHERE id=?`)
	defer stmt.Close()

	ret, err := stmt.Exec("pending",gasprice.Int64(),transferhash,id)
	if err != nil {
		fmt.Printf("UpdatePending error: %v\n", err)
		return err
	}
	if _, err := ret.RowsAffected(); nil != err {
		//fmt.Println("RowsAffected:", RowsAffected)
		return err
	}
	return nil
}

func (self *StoreState)UpdateTransferedFailed(transferhash string,gas big.Int) error{
	stmt, _ := self.db.Prepare(`UPDATE callinfo SET status=?,gas=?,transferedtime=now() WHERE transferhash=?`)
	defer stmt.Close()

	ret, err := stmt.Exec("transfered",gas.Int64(),transferhash)
	if err != nil {
		fmt.Printf("UpdateTransfered error: %v\n", err)
		return err
	}
	if _, err := ret.RowsAffected(); nil != err {
		//fmt.Println("RowsAffected:", RowsAffected)
		return err
	}
	return nil
}

func (self *StoreState)UpdateTransfered(transferhash string,gas big.Int,blocknumber big.Int,logdata string,logtopics string) error{
	stmt, _ := self.db.Prepare(`UPDATE callinfo SET status=?,gas=?,blocknumber=?,logdata=?,logtopics=?,transferedtime=now() WHERE transferhash=?`)
	defer stmt.Close()

	ret, err := stmt.Exec("transfered",gas.Int64(),blocknumber.Int64(),logdata,logtopics,transferhash)
	if err != nil {
		fmt.Printf("UpdateTransfered error: %v\n", err)
		return err
	}
	if _, err := ret.RowsAffected(); nil != err {
		//fmt.Println("RowsAffected:", RowsAffected)
		return err
	}
	return nil
}

func (self *StoreState)UpdateFinished(finishblock big.Int,blockcount int) error{
	stmt, _ := self.db.Prepare(`UPDATE callinfo SET status=?,finishblock=?,finishedtime=now() WHERE blocknumber+?<=? AND status = ?`)
	defer stmt.Close()

	ret, err := stmt.Exec("finished",finishblock.Int64(),blockcount,finishblock.Int64(),"transfered")
	if err != nil {
		fmt.Printf("UpdateFinished error: %v\n", err)
		return err
	}
	if _, err := ret.RowsAffected(); nil != err {
		//fmt.Println("RowsAffected:", RowsAffected)
		return err
	}

	stmt1, _ := self.db.Prepare(`UPDATE incoming SET status=?,finishblock=?,finishedtime=now() WHERE blocknumber+?<=? AND status = ?`)
	defer stmt1.Close()

	ret1, err := stmt1.Exec("finished",finishblock.Int64(),blockcount,finishblock.Int64(),"init")
	if err != nil {
		fmt.Printf("UpdateFinished1 error: %v\n", err)
		return err
	}
	if _, err := ret1.RowsAffected(); nil != err {
		//fmt.Println("RowsAffected:", RowsAffected)
		return err
	}

	return nil
}



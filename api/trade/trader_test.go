package trade

import (
	"bountyHunter/util"
	"log"
	"testing"
)

func newAccount() *Account {
	account := &Account{}
	err := util.YamlFileDecode(util.GetBasePath()+"/config/account.yaml", account)
	if err != nil {
		panic(err)
	}
	account.Login()
	return account
}

func Test_Login(t *testing.T) {
	a := newAccount()
	a.Login()
}

func Test_Postion(t *testing.T) {
	a := newAccount()
	log.Println(a.Position())
}

func Test_Buy(t *testing.T) {
	a := newAccount()
	id, err := a.Buy("150260", 1.430, 100)
	log.Println(id, err)
}

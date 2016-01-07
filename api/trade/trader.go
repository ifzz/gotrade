package trade

import (
	"bountyHunter/util"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/axgle/mahonia"
	"github.com/mreiferson/go-httpclient"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/cookiejar"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type StockPosition struct {
	Code            string `yaml:"-"`
	Name            string `yaml:"-"`
	Amount          int64  `yaml:"amount"`
	AvailableAmount int64  `yaml:"available_amount"`
	FrozenAmount    int64  `yaml:"frozen_amount"`
}

type Balance struct {
	Balance          float64
	AvailableBalance float64
	FrozenBalance    float64
}

type Order struct {
	Code   string
	Name   string
	Amount int64
	Price  float64
	Id     int64
	Type   string
}

type Account struct {
	client    *http.Client
	logger    *logrus.Logger
	Fee       float64 `yaml:"fee"`
	Uid       string  `yaml:"uid"`
	Username  string  `yaml:"username"`
	Account1  string  `yaml:"account1"`
	Account2  string  `yaml:"account2"`
	Password1 string  `yaml:"password1"`
	Password2 string  `yaml:"password2"`
	Password3 string  `yaml:"password3"`
	baseUrl   string
}

type Data struct {
	No string `json:"entrust_no"`
}

type Result struct {
	ErrorCode    string `json:"cssweb_code"`
	ErrorMessage string `json:"cssweb_msg"`
	Item         []Data `json:"item"`
}

// 登录
func (account *Account) Login() (err error) {
	cookieJar, _ := cookiejar.New(nil)
	account.logger = util.NewLogger("trader")
	transport := &httpclient.Transport{
		ConnectTimeout:        3 * time.Second,
		RequestTimeout:        3 * time.Second,
		ResponseHeaderTimeout: 3 * time.Second,
	}
	defer transport.Close()
	account.client = &http.Client{
		CheckRedirect: nil,
		Jar:           cookieJar,
		Transport:     transport,
	}
	account.logger.Info("begin login")
	account.baseUrl = "https://tradegw.htsc.com.cn/?"
	cacheByte, _ := ioutil.ReadFile(util.GetBasePath() + "/cache/" + account.Username + "Uid")
	cacheUid := string(cacheByte)
	if cacheUid != "" {
		account.logger.Info("read cache uid : " + cacheUid)
		account.Uid = cacheUid
		return
	}
	account.logger.Info("get verfiy code")
	loginUrl := "https://service.htsc.com.cn/service/login.jsp"
	req, _ := http.NewRequest("GET", loginUrl, nil)
	resp, _ := account.client.Do(req)
	req, _ = http.NewRequest("GET", "https://service.htsc.com.cn/service/pic/verifyCodeImage.jsp", nil)
	resp, _ = account.client.Do(req)
	defer resp.Body.Close()
	image, _ := ioutil.ReadAll(resp.Body)
	ioutil.WriteFile(util.GetBasePath()+"/cache/verify.jpg", image, 0644)
	var code string
	fmt.Println("input code:")
	fmt.Scanf("%s", &code)
	var raw = fmt.Sprintf("userType=jy&loginEvent=1&trdpwdEns=%s&macaddr=08-00-27-CE-7E-3E&hddInfo=VB0088e34c-9198b670+&lipInfo=10.0.2.15+&topath=null&accountType=1&userName=%s&servicePwd=%s&trdpwd=%s&vcode=", account.Password1, account.Username, account.Password2, account.Password1)
	account.logger.Infof("login post code : %s raw : %s", code, raw)
	req, _ = http.NewRequest("POST", "https://service.htsc.com.cn/service/loginAction.do?method=login", strings.NewReader(raw+code))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Refer", "https://service.htsc.com.cn/service/login.jsp?logout=yes")
	req.Header.Add("User-Agent", "Mozilla/4.0 (compatible; MSIE 8.0; Windows NT 6.1; Trident/4.0; SLCC2; .NET CLR 2.0.50727; .NET4.0C; .NET4.0E)")
	os.Remove("./cache/verify.jpg")
	resp, _ = account.client.Do(req)
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	account.logger.Info("try to get uid")
	req, _ = http.NewRequest("GET", "https://service.htsc.com.cn/service/flashbusiness_new3.jsp?etfCode=", nil)
	resp, _ = account.client.Do(req)
	body, _ = ioutil.ReadAll(resp.Body)
	re := regexp.MustCompile(`var\ data\ =\ "(.+?)"`)
	result := re.FindAllStringSubmatch(string(body), 1)
	data, _ := base64.StdEncoding.DecodeString(result[0][1])
	type User struct {
		Uid string `json:"uid"`
	}
	user := User{}
	json.Unmarshal([]byte(data), &user)
	account.Uid = user.Uid
	account.logger.Info("get uid success" + user.Uid)
	if user.Uid == "" {
		account.logger.Error("login error")
		return errors.New("login error")
	}
	ioutil.WriteFile("./cache/"+account.Username+"Uid", []byte(user.Uid), 0644)
	return
}

// 定时刷新使UID不过期
func (account *Account) RefreshUid() {
	go func() {
		for {
			log.Println("use uid")
			_, err := account.Position()
			if err != nil {
				log.Println("uid maybe out of date: ", err)
			}
			time.Sleep(time.Second * 5)
		}
	}()
	select {}
}

// 异步挂单买
func (account *Account) Buy(stock string, price float64, amount int64) (id int64, err error) {
	price = util.Round(price, 3)
	url := "uid=%s&cssweb_type=STOCK_BUY&version=1&custid=%s&op_branch_no=36&branch_no=36&op_entrust_way=7&op_station=IP$171.212.136.167;MAC$08-00-27-74-30-E4;HDD$VB95a57881-8897b350 &function_id=302&fund_account=%s&password=%s&identity_type=&exchange_type=%s&stock_account=%s&stock_code=%s&entrust_amount=%d&entrust_price=%.3f&entrust_prop=0&entrust_bs=1&ram=0.9656887338496745"
	if substr(stock, 0, 2) == "15" || substr(stock, 0, 2) == "00" || substr(stock, 0, 2) == "30" {
		url = fmt.Sprintf(url, account.Uid, account.Username, account.Username, account.Password3, "2", account.Account1, stock, amount, price)
	} else {
		url = fmt.Sprintf(url, account.Uid, account.Username, account.Username, account.Password3, "1", account.Account2, stock, amount, price)
	}
	account.logger.Infof("begin buy %s %f %d", stock, price, amount)
	url = account.baseUrl + account.base64encode(url)
	req, _ := http.NewRequest("GET", url, nil)
	resp, err := account.client.Do(req)
	if err != nil {
		return
	}
	body, _ := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	jsonString := account.base64decode(string(body))
	result := Result{}
	account.logger.Infof("buy result %s", jsonString)
	json.Unmarshal([]byte(jsonString), &result)
	if result.ErrorMessage == "请重新登录" {
		account.logger.Error("buy token error")
		log.Println("token error")
		account.clearCache()
	}
	if result.ErrorCode != "success" {
		log.Printf("buy error %v", result)
		account.logger.Errorf("buy error %s", result)
		return 0, errors.New("buy error")
	}
	no, _ := strconv.ParseInt(result.Item[0].No, 64)
	account.logger.Infof("buy success op id : %d", no)
	return no, nil
}

// 异步挂单卖
func (account *Account) Sell(stock string, price float64, amount int64) (id int64, err error) {
	price = util.Round(price, 3)
	// todo
	url := "uid=%s&cssweb_type=STOCK_SALE&version=1&custid=%s&op_branch_no=36&branch_no=36&op_entrust_way=7&op_station=IP$171.212.136.167;MAC$08-00-27-74-30-E4;HDD$VB95a57881-8897b350 &function_id=302&fund_account=%s&password=%s&identity_type=&exchange_type=%s&stock_account=%s&stock_code=%s&entrust_amount=%d&entrust_price=%.3f&entrust_prop=0&entrust_bs=2&ram=0.7360913073644042"
	if substr(stock, 0, 2) == "15" || substr(stock, 0, 2) == "00" {
		url = fmt.Sprintf(url, account.Uid, account.Username, account.Username, account.Password3, "2", account.Account1, stock, amount, price)
	} else {
		url = fmt.Sprintf(url, account.Uid, account.Username, account.Username, account.Password3, "1", account.Account2, stock, amount, price)
	}
	account.logger.Infof("begin sell %s %f %d", stock, price, amount)
	url = account.baseUrl + account.base64encode(url)
	req, _ := http.NewRequest("GET", url, nil)
	resp, err := account.client.Do(req)
	if err != nil {
		return
	}
	body, _ := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	jsonString := account.base64decode(string(body))
	result := Result{}
	account.logger.Infof("sell result %s", jsonString)
	json.Unmarshal([]byte(jsonString), &result)
	if result.ErrorMessage == "请重新登录" {
		account.logger.Error("sell token error")
		log.Println("token error")
		account.clearCache()
	}
	if result.ErrorCode != "success" {
		log.Printf("sell error %v", result)
		account.logger.Errorf("sell error %s", result)
		return 0, errors.New("sell error")
	}
	no, _ := strconv.ParseInt(result.Item[0].No, 64)
	account.logger.Infof("sell success op id: %d", no)
	return no, nil
}

// 取消订单
func (account *Account) Cancel(id int64) (err error) {
	url := "uid=%s&cssweb_type=STOCK_CANCEL&version=1&custid=%s&op_branch_no=36&branch_no=36&op_entrust_way=7&op_station=IP$171.212.136.167;MAC$08-00-27-74-30-E4;HDD$VB95a57881-8897b350 &function_id=304&fund_account=%s&password=%s&identity_type=&batch_flag=0&exchange_type=&entrust_no=%d&ram=0.544769384432584"
	url = fmt.Sprintf(url, account.Uid, account.Username, account.Username, account.Password3, id)
	account.logger.Infof("begin cancel %d", id)
	url = account.baseUrl + account.base64encode(url)
	req, _ := http.NewRequest("GET", url, nil)
	resp, _ := account.client.Do(req)
	body, _ := ioutil.ReadAll(resp.Body)
	jsonString := account.base64decode(string(body))
	result := Result{}
	json.Unmarshal([]byte(jsonString), &result)
	if result.ErrorMessage == "请重新登录" {
		log.Println("token error")
		account.logger.Error("cancel token error")
		account.clearCache()
		err = errors.New("token error")
		return
	}
	if result.ErrorCode != "success" {
		log.Printf("cancel error %v", result)
		account.logger.Errorf("cancel error %s", result)
		return errors.New("cancel error")
	}
	no, _ := strconv.ParseInt(result.Item[0].No, 64)
	account.logger.Infof("cancel success op id %d", no)
	return
}

// 获取持仓
func (account *Account) Position() (data []StockPosition, err error) {
	raw := fmt.Sprintf("uid=%s&cssweb_type=GET_STOCK_POSITION&version=1&custid=%s&op_branch_no=36&branch_no=36&op_entrust_way=7&op_station=IP$171.212.136.167;MAC$08-00-27-74-30-E4;HDD$VB95a57881-8897b350 &function_id=403&fund_account=%s&password=%s&identity_type=&exchange_type=&stock_account=&stock_code=&query_direction=&query_mode=0&request_num=100&position_str=&ram=0.39408391434699297",
		account.Uid, account.Username, account.Username, account.Password3)
	param := account.base64encode(raw)
	url := fmt.Sprintf("https://tradegw.htsc.com.cn/?%s", param)
	req, _ := http.NewRequest("GET", url, nil)
	resp, err := account.client.Do(req)
	if err != nil {
		account.logger.Errorln("get position err", err)
		return
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	jsonString := account.base64decode(string(body))
	type Item struct {
		Code            string `json:"stock_code"`
		Name            string `json:"stock_name"`
		Amount          string `json:"current_amount"`
		AvailableAmount string `json:"enable_amount"`
		FrozenAmount    string `json:"hand_flag"`
	}
	type Message struct {
		Code         string `json:"cssweb_code"`
		ErrorMessage string `json:"cssweb_msg"`
		Items        []Item `json:"item"`
	}
	message := Message{}
	json.Unmarshal([]byte(jsonString), &message)
	if message.ErrorMessage == "请重新登录" {
		log.Println("token error")
		account.logger.Error("position token error")
		account.clearCache()
		err = errors.New("token error")
		return
	}
	if len(message.Items) > 1 {
		message.Items = message.Items[:len(message.Items)-1]
		for _, item := range message.Items {
			stockPosition := StockPosition{}
			stockPosition.Name = item.Name
			stockPosition.Code = item.Code
			stockPosition.Amount, _ = strconv.ParseInt(item.Amount, 64)
			stockPosition.AvailableAmount, _ = strconv.ParseInt(item.AvailableAmount, 64)
			stockPosition.FrozenAmount = stockPosition.Amount - stockPosition.AvailableAmount
			data = append(data, stockPosition)
		}
	}
	return
}

// 获取账户资金
func (account *Account) Balance() (data Balance, err error) {
	raw := fmt.Sprintf("uid=%s&cssweb_type=GET_FUNDS&version=1&custid=%s&op_branch_no=36&branch_no=36&op_entrust_way=7&op_station=IP$171.212.136.167;MAC$08-00-27-74-30-E4;HDD$VB95a57881-8897b350 &function_id=405&fund_account=%s&password=%s&identity_type=&money_type=&ram=0.5080185956321657",
		account.Uid, account.Username, account.Username, account.Password3)
	param := base64.StdEncoding.EncodeToString([]byte(raw))
	url := fmt.Sprintf("https://tradegw.htsc.com.cn/?%s", param)
	req, _ := http.NewRequest("GET", url, nil)
	resp, _ := account.client.Do(req)
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	jsonString := account.base64decode(string(body))
	type Item struct {
		Balance          string `json:"current_balance"`
		AvailableBalance string `json:"enable_balance"`
	}
	type Message struct {
		Code         string `json:"cssweb_code"`
		ErrorMessage string `json:"cssweb_msg"`
		Item         []Item `json:"item"`
	}
	message := Message{}
	json.Unmarshal([]byte(jsonString), &message)
	if message.ErrorMessage == "请重新登录" {
		log.Println("token error")
		account.clearCache()
		err = errors.New("token error")
		return
	}
	data.Balance, _ = strconv.ParseFloat(message.Item[0].Balance, 64)
	data.AvailableBalance, _ = strconv.ParseFloat(message.Item[0].AvailableBalance, 64)
	data.FrozenBalance = data.Balance - data.AvailableBalance
	return
}

// 获取未交易成功列表
func (account *Account) Pending() (data []Order, err error) {
	raw := fmt.Sprintf("uid=%s&cssweb_type=GET_CANCEL_LIST&version=1&custid=%s&op_branch_no=36&branch_no=36&op_entrust_way=7&op_station=IP$171.212.137.45;MAC$08-00-27-74-30-E4;HDD$VB95a57881-8897b350 &function_id=401&fund_account=%s&password=%s&identity_type=&exchange_type=&stock_account=&stock_code=&locate_entrust_no=&query_direction=&sort_direction=0&request_num=100&position_str=&ram=0.1524588279426098",
		account.Uid, account.Username, account.Username, account.Password3)
	param := base64.StdEncoding.EncodeToString([]byte(raw))
	url := fmt.Sprintf("https://tradegw.htsc.com.cn/?%s", param)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		account.logger.Errorln("get pending err", err)
		return
	}
	resp, err := account.client.Do(req)
	if err != nil {
		account.logger.Errorln("get pending err", err)
		return
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	jsonString := account.base64decode(string(body))
	type Item struct {
		Code   string `json:"stock_code"`
		Name   string `json:"stock_name"`
		Amount string `json:"entrust_amount"`
		Price  string `json:"entrust_price"`
		Id     string `json:"entrust_no"`
		Type   string `json:"entrust_bs"`
	}
	type Message struct {
		Code         string `json:"cssweb_code"`
		ErrorMessage string `json:"cssweb_msg"`
		Items        []Item `json:"item"`
	}
	message := Message{}
	json.Unmarshal([]byte(jsonString), &message)
	if message.ErrorMessage == "请重新登录" {
		log.Println("token error")
		account.clearCache()
		err = errors.New("token error")
		return
	}
	if len(message.Items) == 0 {
		return
	}
	message.Items = message.Items[:len(message.Items)-1]
	for _, item := range message.Items {
		order := Order{}
		order.Name = item.Name
		order.Code = item.Code
		order.Amount, _ = strconv.ParseInt(item.Amount, 64)
		order.Price, _ = strconv.ParseInt(item.Price, 64)
		order.Id, _ = strconv.ParseInt(item.Id, 64)
		if item.Type == "2" {
			order.Type = "sell"
		} else {
			order.Type = "buy"
		}
		data = append(data, order)
	}
	return
}

// // 同步买下单后自动检测是否交易成功
// func (account *Account) BuySync(stock string, price float64, amount int64) (err error) {
// 	return
// 	var id int64
// 	id, err = account.Buy(stock, price, amount)
// 	if err != nil {
// 		return
// 	}
// 	log.Println("buy sync order id is ", id)
// 	isDeal := false
// 	for !isDeal {
// 		isDeal = true
// 		log.Println("sync buy not deal wait...")
// 		orderList, err := account.Pending()
// 		for err != nil {
// 			orderList, err = account.Pending()
// 		}
// 		for _, order := range orderList {
// 			if order.Id == id {
// 				isDeal = false
// 				break
// 			}
// 		}
// 	}
// 	return
// }

// // 同步卖
// func (account *Account) SellSync(stock string, price float64, amount int64) (err error) {
// 	return
// 	var id int64
// 	id, err = account.Sell(stock, price, amount)
// 	if err != nil {
// 		return
// 	}
// 	log.Println("sell sync order id is ", id)

// 	isDeal := false
// 	for !isDeal {
// 		isDeal = true
// 		log.Println("sync sell not deal wait...")
// 		orderList, err := account.Pending()
// 		for err != nil {
// 			orderList, err = account.Pending()
// 		}
// 		for _, order := range orderList {
// 			if order.Id == id {
// 				isDeal = false
// 				break
// 			}
// 		}
// 	}
// 	return
// }

func (account *Account) clearCache() {
	os.Remove(util.GetBasePath() + "/cache/" + account.Username + "Uid")
}

func (account *Account) base64decode(str string) string {
	data, _ := base64.StdEncoding.DecodeString(str)
	str = fmt.Sprintf("%s", data)
	enc := mahonia.NewDecoder("gbk")
	gbk := enc.ConvertString(str)
	gbk = strings.Replace(gbk, "\n", "", -1)
	return gbk
}

func (account *Account) base64encode(str string) string {
	return base64.StdEncoding.EncodeToString([]byte(str))
}

func substr(s string, pos, length int) string {
	runes := []rune(s)
	l := pos + length
	if l > len(runes) {
		l = len(runes)
	}
	return string(runes[pos:l])
}

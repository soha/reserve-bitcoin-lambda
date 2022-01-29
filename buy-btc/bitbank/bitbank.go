package bitbank

import (
	"buy-btc/utils"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
	"strconv"
	"time"
)

const baseURL = "https://public.bitbank.cc"
const basePrivateURL = "https://api.bitbank.cc"
const pair = "pair"
const MinimumAmount = 0.0001 //bitbankにおけるETHの最小注文数量
const BtcPlace = 5.0

type APIClient struct {
	apiKey    string
	apiSecret string
}

func NewAPIClient(apiKey, apiSecret string) *APIClient {
	return &APIClient{apiKey, apiSecret}
}

func GetTicker(ch chan *Ticker, errCh chan error) {
	pair := "eth_jpy"
	url := baseURL + "/" + pair + "/ticker"
	res, err := utils.DoHttpRequest("GET", url, nil, nil, nil)
	if err != nil {
		ch <- nil
		errCh <- err
		return
	}

	var ticker Ticker
	err = json.Unmarshal(res, &ticker)
	if err != nil {
		ch <- nil
		errCh <- err
		return
	}

	ch <- &ticker
	errCh <- nil
}

/*
func PlaceOrderWithParams(client *APIClient, price, size float64) (*OrderRes, error) {
	order := Order{
		ProductCode:     Btcjpy.String(),
		ChildOrderType:  Limit.String(),
		Side:            Buy.String(),
		Price:           price,
		Size:            size,
		MinuteToExpires: 4320, //3days
		TimeInForce:     Gtc.String(),
	}

	orderRes, err := client.PlaceOrder(&order)
	if err != nil {
		return nil, err
	}

	return orderRes, nil
}

//ロジックを取得する関数 →　高階関数
func GetBuyLogic(strategy int) func(float64, *Ticker) (float64, float64) {
	var logic func(float64, *Ticker) (float64, float64)

	//TODO: ロジックを任意で追加
	switch strategy {
	case 1:
		//LTPの98.5%の価格
		logic = func(budget float64, t *Ticker) (float64, float64) {
			var buyPrice, buySize float64
			buyPrice = utils.RoundDecimal(t.Ltp * 0.985)
			buySize = utils.CalcAmount(buyPrice, budget, btcMinimumAmount, btcPlace)

			//TODO: 他の通貨ペア追加時にはif文での判定を行うこと
			//if t.ProductCode == Btcjpy.String() {
			//	xxxxxx
			//} else if t.ProductCode == Ethjpy.String() {
			//	xxxxxx
			//}
			return buyPrice, buySize
		}
		break
	default:
		//BestASKを注文価格とする
		logic = func(budget float64, t *Ticker) (float64, float64) {
			var buyPrice, buySize float64
			buyPrice = utils.RoundDecimal(t.BestAsk)
			buySize = utils.CalcAmount(buyPrice, budget, btcMinimumAmount, btcPlace)
			return buyPrice, buySize
		}
		break
	}

	return logic
}
*/

//成行注文
func MarketOrder(client *APIClient, amount float64) (*OrderRes, error) {
	order := Order{
		Pair:   "eth_jpy",
		Side:   "buy",
		Type:   "market",
		Amount: strconv.FormatFloat(amount, 'f', 4, 64), //bitbankのETHの最小取引数量は、0.0001 なので少数4桁まで
	}
	log.Println("order")
	log.Println(order)
	orderRes, err := client.PlaceOrder(&order)
	if err != nil {
		return nil, err
	}

	return orderRes, nil
}

//新規注文を出すfunction
func (client *APIClient) PlaceOrder(order *Order) (*OrderRes, error) {
	method := "POST"
	path := "/v1/user/spot/order"
	url := basePrivateURL + path
	data, err := json.Marshal(order)
	if err != nil {
		return nil, err
	}
	log.Println("data")
	log.Println(string(data))

	header := client.getHeader(method, path, data)

	log.Println("url")
	log.Println(url)
	//log.Println("header")
	//log.Println(header) // accesskey,secretkey出るので注意
	res, err := utils.DoHttpRequest(method, url, header, map[string]string{}, data)
	log.Println("res")
	log.Println(string(res))
	if err != nil {
		return nil, err
	}

	var orderRes OrderRes
	err = json.Unmarshal(res, &orderRes)
	if err != nil {
		return nil, err
	}

	if orderRes.Success != 1 {
		return nil, errors.New(string(res))
	}

	return &orderRes, nil
}

func (client *APIClient) getHeader(method, path string, body []byte) map[string]string {
	//timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	//timestamp := strconv.FormatInt(time.Now().UnixNano()/int64(time.Millisecond), 10)
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	nonce := timestamp

	text := nonce + string(body)
	mac := hmac.New(sha256.New, []byte(client.apiSecret))
	mac.Write([]byte(text))
	sign := hex.EncodeToString(mac.Sum(nil))

	return map[string]string{
		"ACCESS-KEY":       client.apiKey,
		"ACCESS-NONCE":     nonce,
		"ACCESS-SIGNATURE": sign,
	}
}

type Ticker struct {
	Success int        `json:"success"`
	Data    TickerItem `json:"data"`
}

type TickerItem struct {
	Sell      string `json:"sell"`
	Buy       string `json:"buy"`
	High      string `json:"high"`
	Low       string `json:"low"`
	Last      string `json:"last"`
	Volume    string `json:"vol"`
	Timestamp int64  `json:"timestamp"`
}

type Order struct {
	Pair   string `json:"pair"`
	Amount string `json:"amount"`
	Side   string `json:"side"`
	Type   string `json:"type"`
	//TimeInForce   string `json:"timeInForce"`
	//Price string `json:"price"`
	//LosscutPrice  string `json:"losscutPrice"`
	//CancelBefore bool   `json:"cancelBefore"`
}

type OrderRes struct {
	Success      int          `json:"success"`
	OrderResData OrderResData `json:"data"`
}

type OrderResData struct {
	OrderID         int64  `json:"order_id"`
	Pair            string `json:"pair"`
	Side            string `json:"side"`
	Type            string `json:"type"`
	StartAmount     string `json:"start_amount"`
	RemainingAmount string `json:"remaining_amount"`
	ExecutedAmount  string `json:"executed_amount"`
	Price           string `json:"price"`
	PostOnly        bool   `json:"post_only"`
	AveragePrice    string `json:"average_price"`
	OrderedAt       int64  `json:"ordered_at"`
	ExpireAt        int64  `json:"expire_at"`
	TriggerPrice    string `json:"trigger_price"`
	Status          string `json:"status"`
}

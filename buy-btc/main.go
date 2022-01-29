package main

import (
	"buy-btc/bitbank"
	"buy-btc/utils"
	"fmt"
	"log"
	"strconv"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ssm"
)

func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	tickerChan := make(chan *bitbank.Ticker)
	errChan := make(chan error)
	defer close(tickerChan)
	defer close(errChan)

	go bitbank.GetTicker(tickerChan, errChan)
	ticker := <-tickerChan
	err := <-errChan
	if err != nil {
		return getErrorResponse(err.Error()), err
	}

	log.Println("ticker:", ticker)

	apiKey, err := getParameter("buy-btc-apikey")
	if err != nil {
		return getErrorResponse(err.Error()), err
	}
	apiSecret, err := getParameter("buy-btc-apisecret")
	if err != nil {
		return getErrorResponse(err.Error()), err
	}
	// 一度に購入する金額(文字列)
	oneShotBuyBudgetStr, err := getParameter("buy-btc-oneshotbuybudget")
	if err != nil {
		return getErrorResponse(err.Error()), err
	}
	//oneShotBuyBudgetStr := "500"

	oneShotBuyBudget, err := strconv.ParseFloat(oneShotBuyBudgetStr, 64)
	if err != nil {
		return getErrorResponse(err.Error()), err
	}
	log.Println("oneShotBuyPrice:", oneShotBuyBudget)

	// 注文価格(自然数)
	buyPrice, err := strconv.ParseFloat(ticker.Data.Last, 64)
	if err != nil {
		return getErrorResponse(err.Error()), err
	}

	buySize := utils.CalcAmount(buyPrice, oneShotBuyBudget, bitbank.MinimumAmount, bitbank.BtcPlace)

	//buyAmount := oneShotBuyPrice / buyPrice
	//buyAmountStr := strconv.FormatFloat(buyAmount, 'f', 4, 64) //bitbankのETHの最小取引数量は、0.0001 なので少数4桁まで
	log.Println("buyAmount:", buySize)

	client := bitbank.NewAPIClient(apiKey, apiSecret)

	orderRes, err := bitbank.MarketOrder(client, buySize) //bitbankのETHの最小取引数量は、0.0001

	if err != nil {
		log.Println(err)
		return events.APIGatewayProxyResponse{
			Body:       "Bad Request!!",
			StatusCode: 400,
		}, nil
	}

	return events.APIGatewayProxyResponse{
		Body:       fmt.Sprintf("res:%+v", orderRes),
		StatusCode: 200,
	}, nil
}

//System Managerからパラメータを取得する関数
func getParameter(key string) (string, error) {
	// SharedConfigEnable → ~/.aws/config
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	svc := ssm.New(sess, aws.NewConfig().WithRegion("ap-northeast-1"))

	params := &ssm.GetParameterInput{
		Name:           aws.String(key),
		WithDecryption: aws.Bool(true),
	}

	res, err := svc.GetParameter(params)
	if err != nil {
		return "", err
	}

	return *res.Parameter.Value, nil
}

func getErrorResponse(message string) events.APIGatewayProxyResponse {
	return events.APIGatewayProxyResponse{
		Body:       message,
		StatusCode: 400,
	}
}

func main() {
	lambda.Start(handler)
}

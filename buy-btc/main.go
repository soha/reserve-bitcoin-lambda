package main

import (
	"buy-btc/gmocoin"
	"fmt"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ssm"
)

func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	//tickerChan := make(chan *bitflyer.Ticker)
	tickerChan := make(chan *gmocoin.Ticker)
	errChan := make(chan error)
	defer close(tickerChan)
	defer close(errChan)

	/**
		go gmocoin.GetTicker(tickerChan, errChan)
		ticker := <-tickerChan
		err := <-errChan
		if err != nil {
			return getErrorResponse(err.Error()), err
		}
		return events.APIGatewayProxyResponse{
			Body:       fmt.Sprintf("ticker:%s", ticker),
			StatusCode: 200,
		}, nil
	}
	**/

	apiKey, err := getParameter("buy-btc-apikey")
	if err != nil {
		return getErrorResponse(err.Error()), err
	}
	apiSecret, err := getParameter("buy-btc-apisecret")
	if err != nil {
		return getErrorResponse(err.Error()), err
	}

	client := gmocoin.NewAPIClient(apiKey, apiSecret)

	orderRes, err := gmocoin.MarketOrder(client, 0.0001) //GMOコインのETHの最小取引数量は、0.01、0.0001なのは最小取引単位、BTCは、0.0001

	if err != nil {
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

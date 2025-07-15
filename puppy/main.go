package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"puppy/sqsUtils"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

// Define a struct for the expected JSON body

type Input struct {
	URL    *string `json:"url"`
	Prio   *int8   `json:"priority"`
	GrpId  *string `json:"groupid"`
	Portal *string `json:"portal"`
}

var (
	env = sqsUtils.LoadEnv()
	svc *sqs.Client
)

func init() {

	var err error
	svc, err = sqsUtils.ConnectToSqs(env.Region)
	if err != nil {
		fmt.Printf("Lamnda setup failed due to failed sqs connection")
	}
}

func main() {

	lambda.Start(handler)
}

func handler(request events.APIGatewayProxyRequest) (interface{}, error) {

	// Unmarshal the JSON body
	var input Input
	err := json.Unmarshal([]byte(request.Body), &input)
	if err != nil {
		return retResp(http.StatusBadRequest, "Something wrong with the Data Posted"), nil
	}

	//validation if json is good
	errlist := input.Validate()
	if len(errlist) > 2 {
		return retResp(http.StatusBadRequest, errlist), nil
	}

	in, _ := json.Marshal(input)

	//send for genration to ec2
	resp, err := http.Post(env.Ec2_api, "application/json", bytes.NewBuffer(in))
	if err != nil {
		return retResp(http.StatusBadRequest, `{"failed":"api failed"}`), nil

	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return retResp(http.StatusOK, string(body)), nil

}

func (in *Input) Validate() string {
	errList := make(map[int8]string)
	if in.URL == nil || *in.URL == "" {
		errList[0] = "Url is not provided"
	}
	if in.Prio == nil {
		errList[1] = "priority not set"
	}
	if *in.Prio == 0 || *in.Prio > 2 {
		errList[2] = "priority accepted : 1 Or 2"
	}
	if in.GrpId == nil || *in.GrpId == "" {
		errList[3] = "groupid not set"
	}
	if in.Portal == nil || *in.Portal == "" {
		errList[4] = "portal is not set"
	}
	errList_str, _ := json.Marshal(errList)
	return string(errList_str)
}

func retResp(statusCode int, Body string) events.APIGatewayProxyResponse {
	return events.APIGatewayProxyResponse{
		StatusCode: statusCode,
		Body:       Body,
	}

}

func ImmediateResponse() {

}

func PushToQueue() {

}

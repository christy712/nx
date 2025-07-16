package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
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
	var (
		input      Input
		StatusCode int
		body       string
	)
	err := json.Unmarshal([]byte(request.Body), &input)
	if err != nil {
		return retResp(http.StatusBadRequest, "Something wrong with the Data Posted"), nil
	}

	//validation if json is good
	errlist, err := input.Validate()
	if err != nil {
		return retResp(http.StatusBadRequest, errlist), nil
	}

	if *input.Prio == 1 {
		StatusCode, body, err = ImmediateResponse(input)
	} else {
		StatusCode, body, err = PushToQueue(*input.GrpId, *input.URL)
	}

	return retResp(StatusCode, string(body)), nil

}

func retResp(statusCode int, Body string) events.APIGatewayProxyResponse {
	return events.APIGatewayProxyResponse{
		StatusCode: statusCode,
		Body:       Body,
	}
}

func ImmediateResponse(input Input) (int, string, error) {
	in, _ := json.Marshal(input)

	//send for genration to ec2
	resp, err := http.Post(env.Ec2_api, "application/json", bytes.NewBuffer(in))
	if err != nil {
		return http.StatusBadRequest, `{"failed":"api failed"}`, err

	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return http.StatusOK, string(body), nil
}

func PushToQueue(Grp string, body string) (int, string, error) {

	_, msgid, err := sqsUtils.SendToSqs(env.QueueUrl, svc, Grp, body)
	if err != nil {
		return http.StatusBadRequest, `{"failed":"api failed to push to sqs "}`, err
	}

	return http.StatusOK, fmt.Sprintf("{\"Success\":\"pushed to sqs:%s\"}", msgid), nil
}
func (in *Input) Validate() (string, error) {
	errList := make(map[int8]string)

	if in.URL == nil || *in.URL == "" {
		errList[0] = "Url is not provided"
	} else {
		_, err := url.ParseRequestURI(*in.URL)
		if err != nil {
			errList[6] = "Crictical: Cannot Parse URL Provided " + *in.URL
		}
	}

	if in.Prio == nil {
		errList[1] = "priority not set"
	} else {
		if *in.Prio == 0 || *in.Prio > 2 {
			errList[2] = "priority accepted : 1 Or 2"
		}
	}
	if in.GrpId == nil || *in.GrpId == "" {
		errList[3] = "groupid not set"
	}
	if in.Portal == nil || *in.Portal == "" {
		errList[4] = "portal is not set"
	}
	errList_str, _ := json.Marshal(errList)

	var err error
	if len(errList_str) > 2 {
		err = fmt.Errorf("Error")
	}
	return string(errList_str), err
}

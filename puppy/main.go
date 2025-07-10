package main

import (
	"encoding/json"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

// Define a struct for the expected JSON body

var input struct {
	URL         string `json:"url"`
	Title       string `json:"Title"`
	Description string `json:"Description"`
}

func main() {

	lambda.Start(handler)
}

func handler(request events.APIGatewayProxyRequest) (interface{}, error) {

	// Unmarshal the JSON body
	err := json.Unmarshal([]byte(request.Body), &input)

	in, _ := json.Marshal(input)
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
			Body:       "Invalid Json",
		}, nil
	}

	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:       "Received URL: " + string(in),
	}, nil
}

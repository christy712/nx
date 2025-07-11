package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

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

	resp, err := http.Post("https://o48t7ifpeg.execute-api.us-east-1.amazonaws.com/generate/con", "application/json", bytes.NewBuffer(in))

	if err != nil {
		//fmt.Println("Error:", err)
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusBadRequest,
			Body:       `{"failed":"api failed"}`,
		}, nil
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Body:       string(body),
	}, nil
}

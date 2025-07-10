package main

import (
	"context"
	"errors"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

var validTokens = map[string]string{
	"m-monitor": "user1",
}

// Help function to generate an IAM policy
func generatePolicy(principalId, effect, resource string) events.APIGatewayCustomAuthorizerResponse {
	authResponse := events.APIGatewayCustomAuthorizerResponse{PrincipalID: principalId}

	if effect != "" && resource != "" {
		authResponse.PolicyDocument = events.APIGatewayCustomAuthorizerPolicy{
			Version: "2012-10-17",
			Statement: []events.IAMPolicyStatement{
				{
					Action:   []string{"execute-api:Invoke"},
					Effect:   effect,
					Resource: []string{resource},
				},
			},
		}
	}

	// Optional output with custom properties of the String, Number or Boolean type.
	// authResponse.Context = map[string]interface{}{
	// 	"stringKey":  "stringval",
	// 	"numberKey":  123,
	// 	"booleanKey": true,
	// }
	return authResponse
}

func checkandverifytoken(event events.APIGatewayCustomAuthorizerRequest) string {

	authHeader := event.AuthorizationToken
	// Basic validation and parsing
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return `unauthorized`
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")
	_, ok := validTokens[token]
	if !ok {
		return `unauthorized`
	}

	return `allow`
}

func handleRequest(ctx context.Context, event events.APIGatewayCustomAuthorizerRequest) (events.APIGatewayCustomAuthorizerResponse, error) {
	token := checkandverifytoken(event)
	switch strings.ToLower(token) {
	case "allow":
		return generatePolicy("user", "Allow", event.MethodArn), nil
	case "deny":
		return generatePolicy("user", "Deny", event.MethodArn), nil
	case "unauthorized":
		return events.APIGatewayCustomAuthorizerResponse{}, errors.New("Unauthorized") // Return a 401 Unauthorized response
	default:
		return events.APIGatewayCustomAuthorizerResponse{}, errors.New("Error: Invalid token")
	}
}

func main() {
	lambda.Start(handleRequest)
}

package main

import (
	"context"
	"errors"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

var validTokens = map[string]string{
	"mission-os": "pass",
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

	// Check if the Authorization header starts with "Bearer "
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return "unauthorized"
	}

	// Extract the token part (after "Bearer ")
	token := strings.TrimPrefix(authHeader, "Bearer ")
	token = strings.TrimSpace(token) // Clean any extra space

	// Validate token
	value, ok := validTokens[token]
	if !ok || value != "pass" {
		return "deny"
	}
	return "allow"
}

func handleRequest(ctx context.Context, event events.APIGatewayCustomAuthorizerRequest) (events.APIGatewayCustomAuthorizerResponse, error) {
	token := checkandverifytoken(event)
	switch strings.ToLower(token) {
	case "allow":
		return generatePolicy("portal", "Allow", event.MethodArn), nil
	case "deny":
		ret := generatePolicy("portal", "Deny", event.MethodArn)
		return ret, nil
	case "unauthorized":
		return events.APIGatewayCustomAuthorizerResponse{}, errors.New("Unauthorized") // Return a 401 Unauthorized response
	default:
		return events.APIGatewayCustomAuthorizerResponse{}, errors.New("Error: Invalid token")
	}
}

func main() {
	lambda.Start(handleRequest)
}

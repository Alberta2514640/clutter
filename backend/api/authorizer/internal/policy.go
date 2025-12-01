package internal

import "github.com/aws/aws-lambda-go/events"

func CreatePolicy(
	principleID string,
	effect string,
	methodArn string,
	context ...map[string]any,
) events.APIGatewayCustomAuthorizerResponse {

	version := "2012-10-17"
	action := []string{"execute-api:Invoke"}
	resource := []string{methodArn}

	var ctx map[string]any
	if len(context) > 0 {
		ctx = context[0]
	}

	return events.APIGatewayCustomAuthorizerResponse{
		PrincipalID: principleID,
		PolicyDocument: events.APIGatewayCustomAuthorizerPolicy{
			Version: version,
			Statement: []events.IAMPolicyStatement{
				{
					Action:   action,
					Effect:   effect,
					Resource: resource,
				},
			},
		},
		Context: ctx,
	}

}

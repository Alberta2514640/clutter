package generic

// LambdaRuntimes owns the Lambda runtimes a diagram may choose, each with the
// handler code upload uses when the user names none. The generator's allow-list
// and code upload both read it; the config panel offers the same list.
//
// The generated code pins hashicorp/aws ~> 5.0, which refuses nodejs24.x and
// python3.14, so they are left out until that pin moves. provided.al2 stays
// because it is the default in the supported_resources seed; AWS deprecated it
// on 2026-07-31 and blocks creating functions on it from 2027-07-29.
var LambdaRuntimes = map[string]string{
	"nodejs22.x":      "index.handler",
	"python3.13":      "lambda_function.lambda_handler",
	"python3.12":      "lambda_function.lambda_handler",
	"provided.al2023": "bootstrap",
	"provided.al2":    "bootstrap",
}

// LambdaRuntimeNames returns the runtimes in LambdaRuntimes.
func LambdaRuntimeNames() []string {
	names := make([]string, 0, len(LambdaRuntimes))
	for name := range LambdaRuntimes {
		names = append(names, name)
	}
	return names
}

package dynago

import (
	"fmt"
	"net/http"
	"strings"
)

// Encapsulates errors coming from amazon/dynamodb
type Error struct {
	Type          AmazonError    // Parsed and mapped down type
	AmazonRawType string         // Raw error type from amazon
	Exception     string         // Exception from amazon
	Message       string         // Raw message from amazon
	Request       *http.Request  // If available, HTTP request
	RequestBody   []byte         // If available, raw request body bytes
	Response      *http.Response // If available, HTTP response
	ResponseBody  []byte         // If available, raw response body bytes
}

func (e *Error) Error() string {
	exception := e.Exception
	if exception == "" {
		exception = e.AmazonRawType
	}
	return fmt.Sprintf("dynago.Error(%d): %s: %s", e.Type, exception, e.Message)
}

// Parse and create the error
func (e *Error) parse(input *inputError) {
	e.AmazonRawType = input.AmazonRawType
	e.Message = input.Message
	parts := strings.Split(e.AmazonRawType, "#")
	if len(parts) >= 2 {
		e.Exception = parts[1]
		if conf, ok := amazonErrorMap[e.Exception]; ok {
			e.Type = conf.mappedError
		}
	}
}

type inputError struct {
	AmazonRawType string `json:"__type"`
	Message       string `json:"message"`
}

/*
AmazonError is an enumeration of error categories that Dynago returns.

There are many more actual DynamoDB errors, but many of them are redundant from
the perspective of application logic; using these mapped-down errors is a handy
way to implement the logic you want without having a really long set of switch
statements.
*/
type AmazonError int

const (
	ErrorUnknown AmazonError = iota

	ErrorConditionFailed        // Conditional put/update failed; condition not met
	ErrorCollectionSizeExceeded // Item collection (local secondary index) too large
	ErrorThroughputExceeded     // Exceeded provisioned throughput for table or shard
	ErrorNotFound               // Resource referenced by key not found
	ErrorInternalFailure        // Internal server error
	ErrorAuth                   // Encapsulates various authorization errors
	ErrorInvalidParameter       // Encapsulates many forms of invalid input errors
	ErrorServiceUnavailable     // Amazon service unavailable
	ErrorThrottling             // Amazon is throttling us, try later
	ErrorResourceInUse          // Tried to create existing table, delete a table in CREATING state, etc.
)

type amazonErrorConfig struct {
	amazonCode     string
	expectedStatus int
	mappedError    AmazonError
}

// This variable is mostly exposed so that we can document how errors are mapped
var AmazonErrors = []amazonErrorConfig{
	{"ConditionalCheckFailedException", 400, ErrorConditionFailed},
	{"ResourceNotFoundException", 400, ErrorNotFound},
	{"InternalFailure", 500, ErrorInternalFailure},
	{"InternalServerError", 500, ErrorInternalFailure},
	{"IncompleteSignature", 400, ErrorAuth},
	{"InvalidParameterCombination", 400, ErrorInvalidParameter},
	{"InvalidParameterValue", 400, ErrorInvalidParameter},
	{"InvalidQueryParameter", 400, ErrorInvalidParameter},
	{"ItemCollectionSizeLimitExceededException", 400, ErrorCollectionSizeExceeded},
	{"MalformedQueryString", 404, ErrorInvalidParameter},
	{"MissingAction", 400, ErrorInvalidParameter},
	{"MissingAuthenticationToken", 403, ErrorAuth},
	{"MissingParameter", 400, ErrorInvalidParameter},
	{"OptInRequired", 403, ErrorAuth},
	{"ProvisionedThroughputExceededException", 400, ErrorThroughputExceeded},
	{"RequestExpired", 400, ErrorAuth},
	{"ResourceInUseException", 400, ErrorResourceInUse},
	{"ServiceUnavailable", 503, ErrorServiceUnavailable},
	{"ValidationError", 400, ErrorInvalidParameter},
	{"ValidationException", 400, ErrorInvalidParameter},
}

var amazonErrorMap map[string]*amazonErrorConfig

func init() {
	amazonErrorMap = make(map[string]*amazonErrorConfig, len(AmazonErrors))
	for i, conf := range AmazonErrors {
		amazonErrorMap[conf.amazonCode] = &AmazonErrors[i]
	}
}

package response

// Business error codes
const (
	CodeSuccess   = 20000
	CodeCreated   = 20100
	CodeUpdated   = 20001
	CodeDeleted   = 20002
	CodeRetrieved = 20003

	CodeBadRequest   = 40000
	CodeParamInvalid = 40001
	CodeInvalidID    = 40002

	CodeUnauthorized    = 40100
	CodeInvalidToken    = 40101
	CodeTokenExpired    = 40102
	CodeInvalidPassword = 40103

	CodeForbidden       = 40300
	CodeAccountNotFound = 40401
	CodeNotFound        = 40400

	CodeConflict         = 40900
	CodeValidationFailed = 42200

	CodeRateLimitExceeded = 42900

	CodeInternalServer = 50000
	CodeInternalError  = 50001
	CodeDatabaseError  = 50002
	CodeMongoDBError   = 50003
	CodeRedisError     = 50004
)

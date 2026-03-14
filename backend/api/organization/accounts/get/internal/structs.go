package internal

type AccountAccessRole struct {
	Id          string  `json:"id"`
	AccountName string  `json:"account_name"`
	RoleArn     *string `json:"role_arn"`
	Status      string  `json:"status"`
}

type AccountAccessRoleIncomplete struct {
	Id          string `json:"id"`
	AccountName string `json:"account_name"`
	Status      string `json:"status"`
}

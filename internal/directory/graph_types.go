package directory

// Graph wire shapes keep Graph's weird names.
type graphUser struct {
	ID                string  `json:"id"`
	UserPrincipalName string  `json:"userPrincipalName"`
	Mail              *string `json:"mail"`
	MailNickname      *string `json:"mailNickname"`
	DisplayName       string  `json:"displayName"`
	GivenName         *string `json:"givenName"`
	Surname           *string `json:"surname"`
	Department        *string `json:"department"`
	AccountEnabled    *bool   `json:"accountEnabled"`
}

type graphGroup struct {
	ID           string  `json:"id"`
	DisplayName  string  `json:"displayName"`
	MailNickname *string `json:"mailNickname"`
	ODataType    string  `json:"@odata.type"`
}

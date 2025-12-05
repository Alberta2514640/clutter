package generic

import (
	"encoding/json"
)

type AuthorizerContextUserData struct {
	Id               string `json:"id"`
	Email            string `json:"email"`
	Name             string `json:"name"`
	PictureUrl       string `json:"pictureUrl"`
	AccountCreatedOn string `json:"accountCreatedOn"`
}

func GetUserDataFromAuthorizerContext(context map[string]any) (AuthorizerContextUserData, error) {

	// Initialize empty AuthorizerContextUserData struct
	userData := AuthorizerContextUserData{}

	// Unmarshal the Auhtorizer Context from map to JSON bytes
	authorizerContextJsonBytes, err := json.Marshal(context)
	if err != nil {
		return userData, err
	}

	// Marshal JSON bytes into the AuthorizerContexxtUserData struct
	err = json.Unmarshal(authorizerContextJsonBytes, &userData)
	if err != nil {
		return userData, err
	}

	return userData, nil

}

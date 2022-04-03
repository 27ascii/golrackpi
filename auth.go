// Copyright 2022 Ralf Geschke. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package golrackpi

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"errors"

	"crypto/sha256"
	b64 "encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/geschke/golrackpi/internal/helper"

	"net/http"
)

const endpointAuthStart string = "/api/v1/auth/start"
const endpointAuthFinish string = "/api/v1/auth/finish"
const endpointAuthCreateSession string = "/api/v1/auth/create_session"

// AuthStartRequestType defines the JSON structure of the first step in the authentication process
type AuthStartRequestType struct {
	Nonce    string `json:"nonce"`
	Username string `json:"username"`
}

// AuthFinishRequestType defines the JSON structure of the second step in the authentication process
type AuthFinishRequestType struct {
	TransactionId string `json:"transactionId"`
	Proof         string `json:"proof"`
}

// AuthCreateSessionType defines the JSON structure of the last step in the authentication process
type AuthCreateSessionType struct {
	TransactionId string `json:"transactionId"`
	Iv            string `json:"iv"`
	Tag           string `json:"tag"`
	Payload       string `json:"payload"`
}

// AuthClient is the library's instance, it contains the configuration settings with SessionId after successful authentication
type AuthClient struct {
	Scheme    string
	Server    string
	Password  string
	SessionId string
}

// New returns a blank AuthClient instance with default http scheme
func New() *AuthClient {
	client := AuthClient{
		Scheme:    "http",
		Server:    "",
		Password:  "",
		SessionId: "",
	}
	return &client
}

// NewWithParameter returns an AuthClient instance.
// It takes an AuthClient structure as parameter, so it's possible to submit all connection settings in one step.
func NewWithParameter(param AuthClient) *AuthClient {
	if param.Scheme == "https" {
		param.Scheme = "https"
	} else {
		param.Scheme = "http"
	}
	client := AuthClient{
		Scheme:   param.Scheme,
		Server:   param.Server,
		Password: param.Password,
	}
	return &client
}

// SetServer sets the IP address or FQDN of the Kostal inverter
func (c *AuthClient) SetServer(server string) {
	c.Server = server
}

// SetServer sets the password for user access of the Kostal inverter
func (c *AuthClient) SetPassword(password string) {
	c.Password = password
}

// SetServer sets the scheme (http or https) of the Kostal inverter
func (c *AuthClient) SetScheme(scheme string) {
	if scheme == "https" {
		scheme = "https"
	} else {
		scheme = "http"
	}
	c.Scheme = scheme
}

// getUrl is a helper function which creates the API URL
func (c *AuthClient) getUrl(request string) string {
	return c.Scheme + "://" + c.Server + request
}

// Login handles the complete authenciation and login process.
// In case of success it returns the session id.
func (c *AuthClient) Login() (string, error) {

	randomString := helper.RandSeq(12)

	//fmt.Println("randomString:", randomString)
	base64String := b64.StdEncoding.EncodeToString([]byte(randomString))
	//fmt.Println("first nonce mit base64:", base64String)

	userName := "user" // default user name of plant owner

	// create JSON request

	startRequest := AuthStartRequestType{
		Nonce:    base64String,
		Username: userName,
	}

	//Convert User to byte using Json.Marshal
	//Ignoring error.
	body, _ := json.Marshal(startRequest)

	//fmt.Println(bytes.NewBuffer(body))
	//fmt.Println(string(body))

	resp, err := http.Post(c.getUrl(endpointAuthStart), "application/json", bytes.NewBuffer(body))

	// An error is returned if something goes wrong
	if err != nil {
		return "", errors.New("could not initiate authentication")
	}
	//Need to close the response stream, once response is read.
	//Hence defer close. It will automatically take care of it.
	defer resp.Body.Close()

	responseBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		//Failed to read response.
		return "", errors.New("could not read authentication response")
	}

	// todo: add reaction to existing servers with wrong result

	responseReader := bytes.NewReader(responseBody)

	var result map[string]interface{}
	json.NewDecoder(responseReader).Decode(&result)
	//fmt.Println(result)

	var serverNonce string = result["nonce"].(string)
	//rounds, _ := strconv.Atoi(result["rounds"].(string))
	rounds := int64(result["rounds"].(float64))
	serverSalt := result["salt"].(string)
	transactionId := result["transactionId"].(string)

	// some magic crypto stuff
	var saltedPassword, clientKey, serverKey, storedKey, clientSignature, serverSignature []byte

	serverSaltDecoded, _ := b64.StdEncoding.DecodeString(serverSalt)
	//fmt.Println("Salt decoded:", serverSaltDecoded)
	//fmt.Println("salt decoded hex", fmt.Sprintf("%x", serverSaltDecoded))

	saltedPassword = helper.GetPBKDF2Hash(c.Password, string(serverSaltDecoded), int(rounds))
	//fmt.Println("Salted Password:", saltedPassword)
	//fmt.Println("salted hex", fmt.Sprintf("%x", saltedPassword))

	clientKey = helper.GetHMACSHA256(saltedPassword, "Client Key")

	serverKey = helper.GetHMACSHA256(saltedPassword, "Server Key")

	storedKey = helper.GetSHA256Hash(clientKey)

	authMessage := fmt.Sprintf("n=%s,r=%s,r=%s,s=%s,i=%d,c=biws,r=%s", userName, startRequest.Nonce, string(serverNonce), string(serverSalt), rounds, string(serverNonce))
	//fmt.Println("authMessage", authMessage)
	// bis hierhin ok

	clientSignature = helper.GetHMACSHA256(storedKey, authMessage)
	serverSignature = helper.GetHMACSHA256(serverKey, authMessage)
	//fmt.Println("clientSignature", clientSignature)
	//fmt.Println("serverSignature", serverSignature)

	clientProof := helper.CreateClientProof(clientSignature, clientKey)
	//fmt.Println("clientProof:", clientProof)
	// Perform step 2 of the authentication

	finishRequest := AuthFinishRequestType{
		TransactionId: transactionId,
		Proof:         clientProof,
	}

	finishRequestBody, _ := json.Marshal(finishRequest)

	//fmt.Println(string(finishRequestBody))

	respFinish, errFinish := http.Post(c.getUrl(endpointAuthFinish), "application/json", bytes.NewBuffer(finishRequestBody))

	// An error is returned if something goes wrong
	if errFinish != nil {
		return "", errors.New("could not initiate authentication finish request")
	}
	//Need to close the response stream, once response is read.
	//Hence defer close. It will automatically take care of it.
	defer respFinish.Body.Close()

	//Check response code, if New user is created then read response.

	responseFinishBody, errFinishBody := ioutil.ReadAll(respFinish.Body)
	if errFinishBody != nil {
		//Failed to read response.
		return "", errors.New("could not read from authentication finish request")
	}

	responseFinishReader := bytes.NewReader(responseFinishBody)

	var resultFinish map[string]interface{}
	json.NewDecoder(responseFinishReader).Decode(&resultFinish)

	// signature and token only set when login was successful
	_, authOkSignature := resultFinish["signature"]
	_, authOkToken := resultFinish["token"]
	if !authOkSignature || !authOkToken {
		return "", errors.New("authentication failed")
	}

	signatureStr := resultFinish["signature"].(string)
	signature, _ := b64.StdEncoding.DecodeString(signatureStr)
	token := resultFinish["token"].(string)

	//fmt.Println("Signature", signature)
	//fmt.Println("hex", fmt.Sprintf("%x", signature))

	//fmt.Println("token", token)

	cmpBytes := bytes.Compare(signature, serverSignature)
	//fmt.Println("compared:", cmpBytes)

	if cmpBytes != 0 {
		//fmt.Println("signature and serverSignature are not equal!")
		return "", errors.New("signature check error")
		//os.Exit(1)

	}

	h := hmac.New(sha256.New, []byte(storedKey))
	// Write Data to it
	h.Write([]byte("Session Key"))
	h.Write([]byte(authMessage))
	h.Write([]byte(clientKey))

	protocolKey := h.Sum(nil)
	//fmt.Println("MAC / protocol key:", protocolKey)
	//fmt.Println("hex", fmt.Sprintf("%x", protocolKey))

	ivNonce, _ := helper.GenerateRandomBytes(16)
	//fmt.Println("iv / random bytes", ivNonce)
	//fmt.Println("hex", fmt.Sprintf("%x", ivNonce))

	block, err := aes.NewCipher(protocolKey)
	if err != nil {
		return "", errors.New("cipher creation error " + err.Error())

	}

	//aesgcm, err := cipher.NewGCM(block)
	//aesgcm, err := cipher.NewGCMWithNonceSize(block, 16)
	// default tag size in Go is 16
	aesgcm, err := cipher.NewGCMWithNonceSize(block, 16)
	if err != nil {

		return "", errors.New("cipher error " + err.Error())
	}

	var tag []byte
	//ciphertext := aesgcm.Seal(ivNonce, ivNonce, []byte(token), nil)
	ciphertext := aesgcm.Seal(nil, ivNonce, []byte(token), nil)
	//fmt.Println("ciphertext:", ciphertext)
	//fmt.Printf("%x\n", ciphertext)
	// golang appends tag at the end of ciphertext, so we have to extract it
	ciphertext, tag = ciphertext[:len(ciphertext)-16], ciphertext[len(ciphertext)-16:]
	//fmt.Println("ciphertext ohne:", ciphertext)
	//fmt.Printf("%x\n", ciphertext)
	//fmt.Println("tag:", tag)
	//fmt.Printf("%x\n", tag)

	createSessionRequest := AuthCreateSessionType{
		TransactionId: transactionId,
		Iv:            b64.StdEncoding.EncodeToString(ivNonce),
		Tag:           b64.StdEncoding.EncodeToString(tag),
		Payload:       b64.StdEncoding.EncodeToString(ciphertext),
	}

	createSessionRequestBody, _ := json.Marshal(createSessionRequest)

	//fmt.Println(string(createSessionRequestBody))

	respCreateSession, errCreateSession := http.Post(c.getUrl(endpointAuthCreateSession), "application/json", bytes.NewBuffer(createSessionRequestBody))

	// An error is returned if something goes wrong
	if errCreateSession != nil {
		return "", errors.New("could not create session")

	}
	//Need to close the response stream, once response is read.
	//Hence defer close. It will automatically take care of it.
	defer respCreateSession.Body.Close()

	//Check response code, if New user is created then read response.

	responseCreateSessionBody, errCreateSessionBody := ioutil.ReadAll(respCreateSession.Body)
	if errCreateSessionBody != nil {
		//Failed to read response.

		return "", errors.New("could not read from create session request")
	}

	//Convert bytes to String and print
	//jsonCreateSessionStr := string(responseCreateSessionBody)
	//fmt.Println("Response CreateSession: ", jsonCreateSessionStr)

	responseCreateSessionReader := bytes.NewReader(responseCreateSessionBody)

	var resultCreateSession map[string]interface{}
	json.NewDecoder(responseCreateSessionReader).Decode(&resultCreateSession)
	//fmt.Println(resultCreateSession)
	sessionId, sessionOk := resultCreateSession["sessionId"] // .(string)
	if !sessionOk {
		return "", errors.New("session id not available")
	}

	c.SessionId = sessionId.(string)
	return c.SessionId, nil

	// see https://stackoverflow.com/questions/68350301/extract-tag-from-cipher-aes-256-gcm-golang

}

// Logout deletes the current session
func (c *AuthClient) Logout() (bool, error) {

	client := http.Client{}

	request, err := http.NewRequest("POST", c.getUrl("/api/v1/auth/logout"), nil)
	if err != nil {
		return false, err
	}

	request.Header.Add("authorization", "Session "+c.SessionId)

	response, errReq := client.Do(request)
	if errReq != nil || response.StatusCode != 200 {
		return false, errors.New("logout error")
	}
	c.SessionId = ""

	return true, nil

}

// Me returns information about the current user
func (c *AuthClient) Me() (map[string]interface{}, error) {
	result := make(map[string]interface{})
	client := http.Client{}

	request, err := http.NewRequest("GET", c.getUrl("/api/v1/auth/me"), nil)
	if err != nil {
		fmt.Println(err)
	}

	request.Header.Add("authorization", "Session "+c.SessionId)

	response, errMe := client.Do(request)
	if errMe != nil {
		return result, errMe

	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return result, err
	}
	var jsonResult interface{}
	errJson := json.Unmarshal(body, &jsonResult)
	if errJson != nil {

		return result, errJson
	}

	m, mOk := jsonResult.(map[string]interface{})

	if !mOk {
		return result, errors.New("could not read response")
	}
	return m, nil

}

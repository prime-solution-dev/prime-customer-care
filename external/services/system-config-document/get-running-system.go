package externalService

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"prime-customer-care/config"
)

type GetRunningSystemConfigRequest struct {
	ConfigCode string `json:"config_code"`
	Count      int    `json:"count"`
	Prefix     string `json:"prefix"`
	Update     bool   `json:"update"`
}

type GetRunningSystemConfigResponse struct {
	ConfigCode string   `json:"config_code"`
	Data       []string `json:"data"`
}

func GetRunningSystemConfig(jsonPayload GetRunningSystemConfigRequest) (GetRunningSystemConfigResponse, error) {

	jsonData, err := json.Marshal(jsonPayload)
	if err != nil {
		return GetRunningSystemConfigResponse{}, errors.New("Error marshaling struct to JSON:")
	}
	reqProduct, err := http.NewRequest("POST", config.GET_RUNNING_SYSTEM_CONFIG_ENDPOINT, bytes.NewBuffer(jsonData))
	if err != nil {
		return GetRunningSystemConfigResponse{}, errors.New("Error parsing #1: " + err.Error())
	}

	reqProduct.Header.Set("Content-Type", "application/json")

	// Create a client and execute the request
	client := &http.Client{}
	resp, err := client.Do(reqProduct)
	if err != nil {
		return GetRunningSystemConfigResponse{}, errors.New("Error parsing #2: " + err.Error())
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return GetRunningSystemConfigResponse{}, fmt.Errorf("failed to read response body: %v", err)
	}
	var config GetRunningSystemConfigResponse
	err = json.Unmarshal(body, &config)
	if err != nil {
		return GetRunningSystemConfigResponse{}, fmt.Errorf("failed to unmarshal response body: %v", err)
	}
	return config, nil
}

func GenerateRunningCodes(runningCode string, count int, update bool) ([]string, error) {
	if count <= 0 {
		return []string{}, nil
	}

	getReq := GetRunningSystemConfigRequest{
		ConfigCode: runningCode,
		Count:      count,
		Update:     update,
	}

	runningConfigResp, err := GetRunningSystemConfig(getReq)
	if err != nil {
		return nil, fmt.Errorf("failed to generate codes: %v", err)
	}

	if len(runningConfigResp.Data) != count {
		return nil, errors.New("failed to get correct number of codes from system config")
	}

	return runningConfigResp.Data, nil
}

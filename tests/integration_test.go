package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"org-structure-api/internal/app/bootstrap"
)

const (
	testContentTypeJSON     = "application/json"
	testDepartmentNameRoot  = "Root Department"
	testDepartmentNameChild = "Child Department"
	testEmployeeFullName    = "John Doe"
	testEmployeePosition    = "Developer"
	testTreeDepthValue      = "2"
	testDeleteModeCascade   = "cascade"

	httpStatusOK        = 200
	httpStatusCreated   = 201
	httpStatusNoContent = 204
	httpStatusNotFound  = 404
	httpStatusConflict  = 409
)

func TestOrganizationLifecycle(testInstance *testing.T) {
	applicationInstance := app.Initialize()
	testServer := httptest.NewServer(applicationInstance.Router)
	defer testServer.Close()

	testClient := &http.Client{}

	rootDepartmentPayload := map[string]interface{}{
		"name":      testDepartmentNameRoot,
		"parent_id": nil,
	}
	rootDepartmentID := createDepartmentWithPayload(testInstance, testClient, testServer.URL, rootDepartmentPayload)

	childDepartmentPayload := map[string]interface{}{
		"name":      testDepartmentNameChild,
		"parent_id": rootDepartmentID,
	}
	childDepartmentID := createDepartmentWithPayload(testInstance, testClient, testServer.URL, childDepartmentPayload)

	employeePayload := map[string]interface{}{
		"full_name": testEmployeeFullName,
		"position":  testEmployeePosition,
	}
	createEmployee(testInstance, testClient, testServer.URL, childDepartmentID, employeePayload)

	verifyTreeStructure(testInstance, testClient, testServer.URL, rootDepartmentID)
	validateCycleProtection(testInstance, testClient, testServer.URL, rootDepartmentID, childDepartmentID)
	deleteDepartmentCascade(testInstance, testClient, testServer.URL, rootDepartmentID)
}

func createDepartmentWithPayload(testInstance *testing.T, testClient *http.Client, serverURL string, payload map[string]interface{}) int {
	requestBody, marshalError := json.Marshal(payload)
	if marshalError != nil {
		testInstance.Fatalf("failed to marshal request body: %v", marshalError)
	}

	httpRequest, requestError := http.NewRequestWithContext(context.Background(), "POST", serverURL+"/departments/", bytes.NewBuffer(requestBody))
	if requestError != nil {
		testInstance.Fatalf("failed to create request: %v", requestError)
	}
	httpRequest.Header.Set("Content-Type", testContentTypeJSON)

	httpResponse, responseError := testClient.Do(httpRequest)
	if responseError != nil {
		testInstance.Fatalf("failed to execute request: %v", responseError)
	}
	defer func(Body io.ReadCloser) {
		closeError := Body.Close()
		if closeError != nil {
		}
	}(httpResponse.Body)

	if httpResponse.StatusCode != httpStatusCreated {
		testInstance.Fatalf("expected status %d, got %d", httpStatusCreated, httpResponse.StatusCode)
	}

	var responseDTO map[string]interface{}
	decodeError := json.NewDecoder(httpResponse.Body).Decode(&responseDTO)
	if decodeError != nil {
		testInstance.Fatalf("failed to decode response: %v", decodeError)
	}

	identifierValue, isIdentifierValid := responseDTO["id"].(float64)
	if !isIdentifierValid {
		testInstance.Fatal("response id is not a valid number")
	}
	return int(identifierValue)
}

func createEmployee(testInstance *testing.T, testClient *http.Client, serverURL string, departmentID int, payload map[string]interface{}) {
	requestBody, marshalError := json.Marshal(payload)
	if marshalError != nil {
		testInstance.Fatalf("failed to marshal employee body: %v", marshalError)
	}

	requestURL := serverURL + "/departments/" + strconv.Itoa(departmentID) + "/employees/"
	httpRequest, requestError := http.NewRequestWithContext(context.Background(), "POST", requestURL, bytes.NewBuffer(requestBody))
	if requestError != nil {
		testInstance.Fatalf("failed to create employee request: %v", requestError)
	}
	httpRequest.Header.Set("Content-Type", testContentTypeJSON)

	httpResponse, responseError := testClient.Do(httpRequest)
	if responseError != nil {
		testInstance.Fatalf("failed to execute employee request: %v", responseError)
	}
	defer func(Body io.ReadCloser) {
		closeError := Body.Close()
		if closeError != nil {
		}
	}(httpResponse.Body)

	if httpResponse.StatusCode != httpStatusCreated {
		testInstance.Fatalf("expected employee creation status %d, got %d", httpStatusCreated, httpResponse.StatusCode)
	}
}

func verifyTreeStructure(testInstance *testing.T, testClient *http.Client, serverURL string, rootID int) {
	requestURL := serverURL + "/departments/" + strconv.Itoa(rootID) + "?depth=" + testTreeDepthValue + "&include_employees=true"
	httpRequest, requestError := http.NewRequestWithContext(context.Background(), "GET", requestURL, nil)
	if requestError != nil {
		testInstance.Fatalf("failed to create tree request: %v", requestError)
	}

	httpResponse, responseError := testClient.Do(httpRequest)
	if responseError != nil {
		testInstance.Fatalf("failed to execute tree request: %v", responseError)
	}
	defer func(Body io.ReadCloser) {
		closeError := Body.Close()
		if closeError != nil {
		}
	}(httpResponse.Body)

	if httpResponse.StatusCode != httpStatusOK {
		testInstance.Fatalf("expected tree status %d, got %d", httpStatusOK, httpResponse.StatusCode)
	}

	var responseDTO map[string]interface{}
	decodeError := json.NewDecoder(httpResponse.Body).Decode(&responseDTO)
	if decodeError != nil {
		testInstance.Fatalf("failed to decode tree response: %v", decodeError)
	}

	childrenValue, isArray := responseDTO["children"].([]interface{})
	if !isArray {
		testInstance.Fatal("response children field is not an array")
	}
	if len(childrenValue) == 0 {
		testInstance.Fatal("expected children in tree, got empty array")
	}
}

func validateCycleProtection(testInstance *testing.T, testClient *http.Client, serverURL string, rootID int, childID int) {
	requestPayload := map[string]interface{}{
		"parent_id": childID,
	}
	requestBody, marshalError := json.Marshal(requestPayload)
	if marshalError != nil {
		testInstance.Fatalf("failed to marshal cycle payload: %v", marshalError)
	}

	requestURL := serverURL + "/departments/" + strconv.Itoa(rootID)
	httpRequest, requestError := http.NewRequestWithContext(context.Background(), "PATCH", requestURL, bytes.NewBuffer(requestBody))
	if requestError != nil {
		testInstance.Fatalf("failed to create cycle request: %v", requestError)
	}
	httpRequest.Header.Set("Content-Type", testContentTypeJSON)

	httpResponse, responseError := testClient.Do(httpRequest)
	if responseError != nil {
		testInstance.Fatalf("failed to execute cycle request: %v", responseError)
	}
	defer func(Body io.ReadCloser) {
		closeError := Body.Close()
		if closeError != nil {
		}
	}(httpResponse.Body)

	if httpResponse.StatusCode != httpStatusConflict {
		testInstance.Fatalf("expected cycle protection status %d, got %d", httpStatusConflict, httpResponse.StatusCode)
	}
}

func deleteDepartmentCascade(testInstance *testing.T, testClient *http.Client, serverURL string, departmentID int) {
	requestURL := serverURL + "/departments/" + strconv.Itoa(departmentID) + "?mode=" + testDeleteModeCascade
	httpRequest, requestError := http.NewRequestWithContext(context.Background(), "DELETE", requestURL, nil)
	if requestError != nil {
		testInstance.Fatalf("failed to create delete request: %v", requestError)
	}

	httpResponse, responseError := testClient.Do(httpRequest)
	if responseError != nil {
		testInstance.Fatalf("failed to execute delete request: %v", responseError)
	}
	defer func(Body io.ReadCloser) {
		closeError := Body.Close()
		if closeError != nil {
		}
	}(httpResponse.Body)

	if httpResponse.StatusCode != httpStatusNoContent {
		testInstance.Fatalf("expected delete status %d, got %d", httpStatusNoContent, httpResponse.StatusCode)
	}

	verificationURL := serverURL + "/departments/" + strconv.Itoa(departmentID)
	verificationRequest, verificationError := http.NewRequestWithContext(context.Background(), "GET", verificationURL, nil)
	if verificationError != nil {
		testInstance.Fatalf("failed to create verification request: %v", verificationError)
	}

	verificationResponse, verificationResponseError := testClient.Do(verificationRequest)
	if verificationResponseError != nil {
		testInstance.Fatalf("failed to execute verification request: %v", verificationResponseError)
	}
	defer func(Body io.ReadCloser) {
		closeError := Body.Close()
		if closeError != nil {
		}
	}(verificationResponse.Body)

	if verificationResponse.StatusCode != httpStatusNotFound {
		testInstance.Fatalf("expected not found status %d after deletion, got %d", httpStatusNotFound, verificationResponse.StatusCode)
	}
}

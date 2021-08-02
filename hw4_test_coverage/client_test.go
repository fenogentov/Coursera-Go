package main

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
)

var testCasesErrors = []struct {
	input    string
	req      SearchRequest
	strError string
}{
	{
		input:    "limit invalid",
		req:      SearchRequest{Limit: -1, Offset: 0, Query: "", OrderField: "", OrderBy: 0},
		strError: "limit must be > 0",
	},
	{
		input:    "offset invalid",
		req:      SearchRequest{Limit: 5, Offset: -1, Query: "", OrderField: "", OrderBy: 0},
		strError: "offset must be > 0",
	},
	{
		input:    "oreder_field invalid",
		req:      SearchRequest{Limit: 0, Offset: 0, Query: "", OrderField: "Data", OrderBy: 0},
		strError: "OrderFeld Data invalid",
	},
	{
		input:    "oreder_by -2",
		req:      SearchRequest{Limit: 0, Offset: 0, Query: "", OrderField: "", OrderBy: -2},
		strError: ErrorBadOrderBy,
	},
	{
		input:    "oreder_by +2",
		req:      SearchRequest{Limit: 0, Offset: 0, Query: "", OrderField: "", OrderBy: -2},
		strError: ErrorBadOrderBy,
	},
}

var testCasesOK = []struct {
	input    string
	req      SearchRequest
	strError string
}{
	{
		input:    "limit OK",
		req:      SearchRequest{Limit: 10, Offset: 0, Query: "", OrderField: "", OrderBy: 0},
		strError: "",
	},
	{
		input:    "offset OK",
		req:      SearchRequest{Limit: 10, Offset: 10, Query: "", OrderField: "", OrderBy: 0},
		strError: "",
	},
	{
		input:    "order_field Name",
		req:      SearchRequest{Limit: 10, Offset: 10, Query: "", OrderField: "Name", OrderBy: 0},
		strError: "",
	},
	{
		input:    "order_field Id",
		req:      SearchRequest{Limit: 10, Offset: 10, Query: "", OrderField: "Id", OrderBy: 0},
		strError: "",
	},
	{
		input:    "order_field Age",
		req:      SearchRequest{Limit: 10, Offset: 10, Query: "", OrderField: "Age", OrderBy: 0},
		strError: "",
	},
}

func SearchServerTimeout(w http.ResponseWriter, r *http.Request) {
	time.Sleep(1500 * time.Millisecond)
}

func SearchServerInternalError(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusInternalServerError)
}

func SearchServerResultBadJSON(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte("No JSON"))
}

func SearchServerErrorBadJSON(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusBadRequest)
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte("No JSON"))
}

func SearchServer(w http.ResponseWriter, r *http.Request) {
	// token from header
	if r.Header.Get("AccessToken") == "" {

		searchError := SearchErrorResponse{ErrorBadToken}
		searchErrorJson, err := json.Marshal(searchError)
		if err != nil {
			return
		}
		w.WriteHeader(http.StatusUnauthorized)
		w.Write(searchErrorJson)

		return
	}

	// read []User from xml
	users, err := readDataXML("dataset.xml")
	if err != nil {
		searchError := SearchErrorResponse{"Internal Server Error:\nerror data file"}
		searchErrorJson, err := json.Marshal(searchError)
		if err != nil {
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		w.Write(searchErrorJson)

		return
	}

	// get params
	params, err := getQueryParams(w, r)
	if err != nil {
		return
	}

	// select [] users
	selectUsers := dataProcessing(params, users)

	// response
	jsonResponse, err := json.Marshal(selectUsers)
	if err != nil {
		http.Error(w, "Internal Server", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonResponse)
}

func TestFindUsersAccesseToken(t *testing.T) { // Accesse Token

	tstServ := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer tstServ.Close()

	searchClient := &SearchClient{
		AccessToken: "",
		URL:         tstServ.URL,
	}

	t.Run("bad access token", func(t *testing.T) {
		_, err := searchClient.FindUsers(SearchRequest{0, 0, "", "", 0})
		if err != nil && !strings.Contains(err.Error(), ErrorBadToken) {
			t.Errorf("expected: %v, got: %v", ErrorBadToken, err)
		}
		if err == nil {
			t.Errorf("expected: %v", ErrorBadToken)
		}
	})

	searchClient = &SearchClient{
		AccessToken: "GoodToken",
		URL:         tstServ.URL,
	}

	t.Run("good access token", func(t *testing.T) {
		_, err := searchClient.FindUsers(SearchRequest{0, 0, "", "", 0})
		if err != nil {
			t.Errorf("Error: %v", err)
		}
	})
}

func TestFindUsers(t *testing.T) {

	tstServ := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer tstServ.Close()

	searchClient := &SearchClient{
		AccessToken: "GoodToken",
		URL:         tstServ.URL,
	}

	for _, testCase := range testCasesOK {

		t.Run(testCase.input, func(t *testing.T) {
			result, err := searchClient.FindUsers(testCase.req)
			if err != nil {
				t.Errorf("Error: %v", err)
			}
			if result == nil {
				t.Errorf("result nil")
			}
		})
	}

	t.Run("no next page", func(t *testing.T) {
		result, err := searchClient.FindUsers(SearchRequest{Limit: 25, Offset: 25, Query: "", OrderField: "", OrderBy: 0})
		if result.NextPage {
			t.Errorf("Expected NextPage == false, got NextPage == true")
		}
		if err != nil {
			t.Errorf("Error: %v", err)
		}
	})

	//limit >25
	t.Run("limit >25 ", func(t *testing.T) {
		result, err := searchClient.FindUsers(SearchRequest{Limit: 30, Offset: 0, Query: "", OrderField: "", OrderBy: 0})
		if err != nil {
			t.Errorf("Error: %v", err)
		}
		if len(result.Users) > 25 {
			t.Errorf("response > 25")
		}
	})
}

func TestFindUsersBadParametr(t *testing.T) { // test params
	// limit <0, offset<0, order_field invalid, order_by invalid
	tstServ := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer tstServ.Close()

	searchClient := &SearchClient{
		AccessToken: "GoodToken",
		URL:         tstServ.URL,
	}

	for _, testCase := range testCasesErrors {

		t.Run(testCase.input, func(t *testing.T) {
			_, err := searchClient.FindUsers(testCase.req)
			if err != nil && !strings.Contains(err.Error(), testCase.strError) {
				t.Errorf("expected: %v, got: %v", testCase.strError, err)
			}
			if err == nil {
				t.Errorf("expected: %v", testCase.strError)
			}
		})
	}
}

func TestFindUsersBadJSON(t *testing.T) {
	tstServRes := httptest.NewServer(http.HandlerFunc(SearchServerResultBadJSON))
	defer tstServRes.Close()

	searchClient := &SearchClient{
		AccessToken: "GoodToken",
		URL:         tstServRes.URL,
	}

	t.Run("FindUsers result bad JSON", func(t *testing.T) {
		_, err := searchClient.FindUsers(SearchRequest{Limit: 0, Offset: 0, Query: "", OrderField: "", OrderBy: 0})
		if err != nil && !strings.Contains(err.Error(), "cant unpack result json") {
			t.Errorf("expected: cant unpack result json..., got: %v", err)
		}
		if err == nil {
			t.Errorf("expected: cant unpack result json...")
		}
	})

	tstServErr := httptest.NewServer(http.HandlerFunc(SearchServerErrorBadJSON))
	defer tstServErr.Close()

	searchClient = &SearchClient{
		AccessToken: "GoodToken",
		URL:         tstServErr.URL,
	}

	t.Run("FindUsers error bad JSON", func(t *testing.T) {
		_, err := searchClient.FindUsers(SearchRequest{Limit: 0, Offset: 0, Query: "", OrderField: "", OrderBy: 0})
		if err != nil && !strings.Contains(err.Error(), "cant unpack error json") {
			t.Errorf("expected: cant unpack error json..., got: %v", err)
		}
		if err == nil {
			t.Errorf("expected: cant unpack error json...")
		}
	})
}

func TestFindUsersTimeout(t *testing.T) {
	tstServ := httptest.NewServer(http.HandlerFunc(SearchServerTimeout))
	defer tstServ.Close()

	searchClient := &SearchClient{
		AccessToken: "GoodToken",
		URL:         tstServ.URL,
	}

	t.Run("FindUsers Timeout", func(t *testing.T) {
		_, err := searchClient.FindUsers(SearchRequest{Limit: 0, Offset: 0, Query: "", OrderField: "", OrderBy: 0})
		if err != nil && !strings.Contains(err.Error(), "timeout") {
			t.Errorf("expected: timeout..., got: %v", err)
		}
		if err == nil {
			t.Errorf("expected: timeout...")
		}
	})
}

func TestFindUsersUnknownErr(t *testing.T) {

	searchClient := &SearchClient{
		AccessToken: "GoodToken",
		URL:         "",
	}

	t.Run("FindUsers Timeout", func(t *testing.T) {
		_, err := searchClient.FindUsers(SearchRequest{Limit: 0, Offset: 0, Query: "", OrderField: "", OrderBy: 0})
		if err != nil && !strings.Contains(err.Error(), "unknown error") {
			t.Errorf("expected: unknown error..., got: %v", err)
		}
		if err == nil {
			t.Errorf("expected: unknown error...")
		}
	})
}

func TestFindUsersInternalError(t *testing.T) {

	tstServ := httptest.NewServer(http.HandlerFunc(SearchServerInternalError))
	defer tstServ.Close()

	searchClient := &SearchClient{
		AccessToken: "GoodToken",
		URL:         tstServ.URL,
	}

	t.Run("FindUsers StatusInternalServerError", func(t *testing.T) {
		_, err := searchClient.FindUsers(SearchRequest{Limit: 0, Offset: 0, Query: "", OrderField: "", OrderBy: 0})
		if err != nil && !strings.Contains(err.Error(), "SearchServer fatal error") {
			t.Errorf("expected: SearchServer fatal error, got: %v", err)
		}
		if err == nil {
			t.Errorf("expected: SearchServer fatal error")
		}
	})
}

func readDataXML(fileName string) ([]User, error) {
	xmlFile, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	defer xmlFile.Close()

	byteValue, err := ioutil.ReadAll(xmlFile)
	if err != nil {
		return nil, err
	}

	var xmlUsers Users
	err = xml.Unmarshal(byteValue, &xmlUsers)
	if err != nil {
		return nil, err
	}

	users := make([]User, 0, len(xmlUsers.Row))
	for _, u := range xmlUsers.Row {
		item := User{
			Id:     u.Id,
			Name:   u.FirstName + " " + u.LastName,
			Age:    u.Age,
			About:  u.About,
			Gender: u.Gender,
		}
		users = append(users, item)
	}
	return users, nil

}

func getQueryParams(w http.ResponseWriter, r *http.Request) (*SearchRequest, error) {
	parsQuery := r.URL.Query()

	limit, err := strconv.Atoi(parsQuery.Get("limit"))
	if err != nil {
		searchError := SearchErrorResponse{ErrorBadLimit}
		searchErrorJson, err := json.Marshal(searchError)
		if err != nil {
			return nil, err
		}
		w.WriteHeader(http.StatusBadRequest)
		w.Write(searchErrorJson)
		//		log.Println("limit:", err)
		return nil, err
	}

	offset, err := strconv.Atoi(parsQuery.Get("offset"))
	if err != nil {
		searchError := SearchErrorResponse{ErrorBadOffset}
		searchErrorJson, err := json.Marshal(searchError)
		if err != nil {
			return nil, err
		}
		w.WriteHeader(http.StatusBadRequest)
		w.Write(searchErrorJson)
		//		log.Println("offset:", err)
		return nil, err
	}

	query := parsQuery.Get("query")

	orderField := parsQuery.Get("order_field")
	if orderField != "Id" && orderField != "Age" && orderField != "Name" && orderField != "" {
		searchError := SearchErrorResponse{"ErrorBadOrderField"}
		searchErrorJson, err := json.Marshal(searchError)
		if err != nil {
			return nil, err
		}
		w.WriteHeader(http.StatusBadRequest)
		w.Write(searchErrorJson)
		//		log.Println("order_field: invalid value")
		return nil, fmt.Errorf(ErrorBadOrderField)
	}

	orderBy, err := strconv.Atoi(parsQuery.Get("order_by"))
	if err != nil || orderBy < -1 || orderBy > 1 {
		searchError := SearchErrorResponse{ErrorBadOrderBy}
		searchErrorJson, err := json.Marshal(searchError)
		if err != nil {
			return nil, err
		}
		w.WriteHeader(http.StatusBadRequest)
		w.Write(searchErrorJson)
		//		log.Println("order_by: invalid value")
		return nil, errors.New(ErrorBadOrderBy)
	}

	return &SearchRequest{
		Limit:      limit,
		Offset:     offset,
		Query:      query,
		OrderField: orderField,
		OrderBy:    orderBy,
	}, nil
}

func dataProcessing(params *SearchRequest, users []User) []User {

	// выборка
	selectUsers := selectUsers(params, users)
	// сортировка
	sortUsers := sortUsers(params, selectUsers)
	// обрезка
	offsetUsers := offsetUsers(params, sortUsers)
	limitUsers := limitUsers(params, offsetUsers)

	return limitUsers

}

func offsetUsers(params *SearchRequest, users []User) []User {
	if params.Offset >= len(users) {
		return users
	}

	return users[params.Offset:]
}

func limitUsers(params *SearchRequest, users []User) []User {
	if params.Limit > len(users) || params.Limit == 0 {
		return users
	}

	return users[:params.Limit]
}

func selectUsers(params *SearchRequest, users []User) []User {
	var selectUser []User
	if params.Query == "" {
		selectUser = append(selectUser, users...)

	} else {
		for _, u := range users {
			if strings.Contains(u.About, params.Query) || strings.Contains(u.Name, params.Query) {
				selectUser = append(selectUser, u)
			}
		}
	}

	return selectUser
}

func sortUsers(params *SearchRequest, users []User) []User {
	if params.OrderBy == -1 || params.OrderBy == 1 {
		switch params.OrderField {
		case "Id":
			sort.SliceStable(users, func(i, j int) bool {
				if params.OrderBy == 1 {
					return users[i].Id < users[j].Id
				} else {
					return users[i].Id > users[j].Id
				}
			})
		case "Age":
			sort.SliceStable(users, func(i, j int) bool {
				if params.OrderBy == 1 {
					return users[i].Age < users[j].Age
				} else {
					return users[i].Age > users[j].Age
				}
			})
		case "Name", "":
			sort.SliceStable(users, func(i, j int) bool {
				if params.OrderBy == 1 {
					return users[i].Name < users[j].Name
				} else {
					return users[i].Name > users[j].Name
				}
			})
		}
	}

	return users
}

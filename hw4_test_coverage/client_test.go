package main

import (
	"encoding/json"
	"testing"
	"net/http"
	"net/http/httptest"
	"os"
  "fmt"
	"encoding/xml"
	"io/ioutil"	
	"strconv"
	"time"
)

type UserXML struct {
	Id			int 		`xml:"id"`
	Name 		string	`xml:"first_name"`
	Age			int			`xml:"age"`
	About 	string	`xml:"about"`
	Gender 	string	`xml:"gender"`
}

type Users struct {
	List		[]UserXML `xml:"row"`
}

var UserDataset []UserXML

func TestMain(m *testing.M) {
	setup()
	code := m.Run() 
	os.Exit(code)
}

func setup() {
	file, err := os.Open("./dataset.xml")
	if err != nil {
		panic(err)
	}
	
	dataset, err := ioutil.ReadAll(file)
	if err != nil {
		panic(err)
	}

	u := new(Users)
	xml.Unmarshal(dataset, &u)
	UserDataset = u.List
}

func min(a, b int) int {
	if a < b {
			return a
	}
	return b
}

func TestFindUserSuccess(t *testing.T) {	
	var transmittedQuery string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		transmittedQuery = r.URL.RawQuery

		query := r.URL.Query()		
		offset, _ := strconv.Atoi(query.Get("offset"))
		limit, _ := strconv.Atoi(query.Get("limit"))		
		result, err := json.Marshal(UserDataset[offset:min(offset + limit, len(UserDataset))]	)		
		if err != nil {
			panic(err)
		}
		w.WriteHeader(http.StatusOK)
		w.Write(result)	
	}))

	client := &SearchClient{
		URL: ts.URL,
	}
	response, _ := client.FindUsers(SearchRequest{
		Limit:	 30,
		Offset: 4,
		Query: "abc",
		OrderBy: 1,
		OrderField: "ccc",
	})
	if response.Users[0].Id != UserDataset[4].Id {
		t.Errorf("Incorrect transmitted data")		
	}
	if len(response.Users) != 25 {
		t.Errorf("Invalid default limit")		
	}
	if response.NextPage != true {
		t.Errorf("Invalid NextPage flag")		
	}
	if transmittedQuery != "limit=26&offset=4&order_by=1&order_field=ccc&query=abc" {
		t.Errorf("Invalid query params")		
	}

	response, _ = client.FindUsers(SearchRequest{
		Limit:	5,
		Offset: 30,
	})

	if response.NextPage != false {
		t.Errorf("Invalid NextPage flag")		
	}

	ts.Close()
}

func TestFindUserFailureUnauthorized(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)				
	}))

	client := &SearchClient{
		URL: ts.URL,
	}
	_, err := client.FindUsers(SearchRequest{})
	if err == nil {
		t.Errorf("Should return error if not authorized")
	}

	ts.Close()
}

func TestFindUserFailureInternalServerError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)				
	}))

	client := &SearchClient{
		URL: ts.URL,
	}
	_, err := client.FindUsers(SearchRequest{})
	if err == nil {
		t.Errorf("Should return error if server failed")
	}

	ts.Close()
}

func TestFindUserFailureBadRequest(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, "{\"Error\":\"Something Bad\"}")		
	}))

	client := &SearchClient{
		URL: ts.URL,
	}
	_, err := client.FindUsers(SearchRequest{})
	if err == nil {
		t.Errorf("Should return error")
	}

	ts.Close()
}

func TestFindUserFailureBadRequestInvalidJson(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, "{")		
	}))

	client := &SearchClient{
		URL: ts.URL,
	}
	_, err := client.FindUsers(SearchRequest{})
	if err == nil {
		t.Errorf("Should return error")
	}

	ts.Close()
}

func TestFindUserFailureBadRequestErrorBadOrderField(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, "{\"Error\":\"ErrorBadOrderField\"}")		
	}))

	client := &SearchClient{
		URL: ts.URL,
	}
	_, err := client.FindUsers(SearchRequest{})
	if err == nil {
		t.Errorf("Should return error")	
	}

	ts.Close()
}

func TestFindUserFailureTimeout(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
	}))

	client := &SearchClient{
		URL: ts.URL,
	}
	_, err := client.FindUsers(SearchRequest{})
	if err == nil {
		t.Errorf("Should return error")
	}

	ts.Close()
}

func TestFindUserFailureNetError(t *testing.T) {
	client := &SearchClient{}
	_, err := client.FindUsers(SearchRequest{})
	if err == nil {
		t.Errorf("Should return error")
	}
}

func TestFindUserFailureInvalidJson(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "{")		
	}))

	client := &SearchClient{
		URL: ts.URL,
	}
	_, err := client.FindUsers(SearchRequest{})
	if err == nil {
		t.Errorf("Should return error")
	}

	ts.Close()
}

func TestFindUserChecksLimitAndOffset(t *testing.T) {
	client := &SearchClient{}
	_, err := client.FindUsers(SearchRequest{
		Limit:	 -1,
	})
	if err == nil {
		t.Errorf("Limit should be validated")
	}
	_, err = client.FindUsers(SearchRequest{
		Offset:	 -1,
	})
	if err == nil {
		t.Errorf("Offset should be validated")		
	}
}
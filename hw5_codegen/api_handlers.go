
package main

import (
	"net/http"
	"encoding/json"
	"fmt"
	"strconv"
	"runtime/debug"
	"net/url"
)

type ApiErrorResponse struct {
	Error string `json:"error"`
}

type ApiSuccessResponse struct {
	Error 		string 			`json:"error"`
	Response 	interface{} `json:"response"`
}

var Empty struct{}

func isAuthenticated(r *http.Request) bool {
	return r.Header.Get("X-Auth") == "100500"
}

func errorResponse(status int, message string, w http.ResponseWriter) {
	res, _ := json.Marshal(ApiErrorResponse{message})
	w.WriteHeader(status)
	w.Write(res)
}

func successResponse(status int, obj interface{}, w http.ResponseWriter) {
	res, _ := json.Marshal(ApiSuccessResponse{"", obj})
	w.WriteHeader(status)
	w.Write(res)	
}

func proccessError(err error, w http.ResponseWriter) {	
	switch err.(type) {
	case ApiError:
		errorResponse((err.(ApiError)).HTTPStatus, err.Error(), w)
	default:
		errorResponse(http.StatusInternalServerError, err.Error(), w)
	}
}

func getOrDefault(values url.Values, key string, defaultValue string) string {
	items, ok := values[key]
	if !ok {
		return defaultValue
	}
	if len(items) == 0 {
		return defaultValue
	}

	return items[0]
}


func (h *MyApi) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if err := recover(); err != nil {
			debug.PrintStack()
			fmt.Printf("%#v\n", err)
			errorResponse(http.StatusInternalServerError, "Internal server error", w)
		}
	}()

	switch r.URL.Path {
	
	case "/user/profile":
		h.wrapperProfile(w, r)
	
	case "/user/create":
		h.wrapperCreate(w, r)
		
	default:
		errorResponse(http.StatusNotFound, "unknown method", w)
	}
}

func (h *OtherApi) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if err := recover(); err != nil {
			debug.PrintStack()
			fmt.Printf("%#v\n", err)
			errorResponse(http.StatusInternalServerError, "Internal server error", w)
		}
	}()

	switch r.URL.Path {
	
	case "/user/create":
		h.wrapperCreate(w, r)
		
	default:
		errorResponse(http.StatusNotFound, "unknown method", w)
	}
}



	
func (api *MyApi) wrapperProfile(w http.ResponseWriter, r *http.Request) {
	
	

	r.ParseForm()
	params := new(ProfileParams)
	err := params.fillFromForm(r.Form)
	if err != nil {
		errorResponse(http.StatusBadRequest, err.Error(), w)
		return
	}

	res, err := api.Profile(r.Context(), *params)
	if err != nil {		
		proccessError(err, w)
		return
	}

	successResponse(http.StatusOK, res, w)
}
	
func (api *MyApi) wrapperCreate(w http.ResponseWriter, r *http.Request) {
	
	if r.Method != "POST" {
		errorResponse(http.StatusNotAcceptable, "bad method", w)
		return
	}	
	
	
	ok := isAuthenticated(r)
	if !ok {
		errorResponse(http.StatusForbidden, "unauthorized", w)
		return 
	}
	

	r.ParseForm()
	params := new(CreateParams)
	err := params.fillFromForm(r.Form)
	if err != nil {
		errorResponse(http.StatusBadRequest, err.Error(), w)
		return
	}

	res, err := api.Create(r.Context(), *params)
	if err != nil {		
		proccessError(err, w)
		return
	}

	successResponse(http.StatusOK, res, w)
}
	

	
func (api *OtherApi) wrapperCreate(w http.ResponseWriter, r *http.Request) {
	
	if r.Method != "POST" {
		errorResponse(http.StatusNotAcceptable, "bad method", w)
		return
	}	
	
	
	ok := isAuthenticated(r)
	if !ok {
		errorResponse(http.StatusForbidden, "unauthorized", w)
		return 
	}
	

	r.ParseForm()
	params := new(OtherCreateParams)
	err := params.fillFromForm(r.Form)
	if err != nil {
		errorResponse(http.StatusBadRequest, err.Error(), w)
		return
	}

	res, err := api.Create(r.Context(), *params)
	if err != nil {		
		proccessError(err, w)
		return
	}

	successResponse(http.StatusOK, res, w)
}
	



func (s *ProfileParams) fillFromForm(params url.Values) error {
	
		
	s.Login = getOrDefault(params, "login", "")	
				
		

		
		
	if s.Login == "" {
		return fmt.Errorf("login must me not empty")
	}

		
	

	return nil
}

func (s *CreateParams) fillFromForm(params url.Values) error {
	
		
	s.Login = getOrDefault(params, "login", "")	
				
		

		
		
	if s.Login == "" {
		return fmt.Errorf("login must me not empty")
	}

		
		
	if len(s.Login) < 10 {
		return fmt.Errorf("login len must be >= 10")
	}

		
	
		
	s.Name = getOrDefault(params, "full_name", "")	
				
		

		
	
		
	s.Status = getOrDefault(params, "status", "user")	
				
		

		
		
	StatusValues := map[string]struct{}{
		
			"user"	: Empty,
		
			"moderator"	: Empty,
		
			"admin"	: Empty,
		
	}
	if _, ok := StatusValues[s.Status]; !ok {
		return fmt.Errorf("status must be one of [user, moderator, admin]")
	}
		
	
				
		
	Age, err := strconv.Atoi(getOrDefault(params, "age", ""))
	if err != nil {		
		return fmt.Errorf("age must be int")
	}
	s.Age = Age
		

		
		
	if s.Age < 0 {
		return fmt.Errorf("age must be >= 0")
	}

		
		
	if s.Age > 128 {
		return fmt.Errorf("age must be <= 128")
	}	

		
	

	return nil
}

func (s *OtherCreateParams) fillFromForm(params url.Values) error {
	
		
	s.Username = getOrDefault(params, "username", "")	
				
		

		
		
	if s.Username == "" {
		return fmt.Errorf("username must me not empty")
	}

		
		
	if len(s.Username) < 3 {
		return fmt.Errorf("username len must be >= 3")
	}

		
	
		
	s.Name = getOrDefault(params, "account_name", "")	
				
		

		
	
		
	s.Class = getOrDefault(params, "class", "warrior")	
				
		

		
		
	ClassValues := map[string]struct{}{
		
			"warrior"	: Empty,
		
			"sorcerer"	: Empty,
		
			"rouge"	: Empty,
		
	}
	if _, ok := ClassValues[s.Class]; !ok {
		return fmt.Errorf("class must be one of [warrior, sorcerer, rouge]")
	}
		
	
				
		
	Level, err := strconv.Atoi(getOrDefault(params, "level", ""))
	if err != nil {		
		return fmt.Errorf("level must be int")
	}
	s.Level = Level
		

		
		
	if s.Level < 1 {
		return fmt.Errorf("level must be >= 1")
	}

		
		
	if s.Level > 50 {
		return fmt.Errorf("level must be <= 50")
	}	

		
	

	return nil
}


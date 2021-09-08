package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
)

type Response map[string]interface{}

func sendJsonResponse(w http.ResponseWriter, response interface{}, status int) {
	jsonResponse, _ := json.Marshal(response)
	w.WriteHeader(status)
	w.Write(jsonResponse)
}

func validateRequired(name, value string) error {
	if value == "" {
		return fmt.Errorf("%s must me not empty", name)
	}
	return nil
}

func validateMin(name string, value interface{}, min int) error {
	switch v := value.(type) {
	case string:
		{
			if len([]rune(v)) < min {
				return fmt.Errorf("%s len must be >= %d", name, min)
			}
		}
	case int:
		{
			if v < min {
				return fmt.Errorf("%s must be >= %d", name, min)
			}
		}
	}
	return nil
}

func validateMax(name string, value interface{}, max int) error {
	switch v := value.(type) {
	case string:
		{
			if len([]rune(v)) > max {
				return fmt.Errorf("%s len must be <= %d", name, max)
			}
		}
	case int:
		{
			if v > max {
				return fmt.Errorf("%s must be <= %d", name, max)
			}
		}
	}
	return nil
}

func (srv *MyApi) handlerCreate(w http.ResponseWriter, r *http.Request) {
	// authorization
	if r.Header.Get("X-Auth") != "100500" {
		sendJsonResponse(w, Response{"error": "unauthorized"}, http.StatusForbidden)
		return
	}
	//log.Println("authorization ok")
	login := r.FormValue("login")
	if err := validateRequired("login", login); err != nil {
		sendJsonResponse(w, Response{"error": err.Error()}, http.StatusBadRequest)
		return
	}
	if err := validateMin("login", login, 10); err != nil {
		sendJsonResponse(w, Response{"error": err.Error()}, http.StatusBadRequest)
		return
	}
	//	log.Println("login ok")
	name := r.FormValue("full_name")
	//	log.Println("full_name ok")
	lStatus := map[string]struct{}{
		"user":      {},
		"moderator": {},
		"admin":     {},
	}
	status := r.FormValue("status")
	if status == "" {
		status = "user"
	}
	_, ok := lStatus[status]
	if !ok {
		sendJsonResponse(w, Response{"error": "status must be one of [user, moderator, admin]"}, http.StatusBadRequest)
		return
	}
	//	log.Println("status ok")
	age, err := strconv.Atoi(r.FormValue("age"))
	if err != nil {
		sendJsonResponse(w, Response{"error": "age must be int"}, http.StatusBadRequest)
		return
	}
	if err := validateMin("age", age, 0); err != nil {
		sendJsonResponse(w, Response{"error": err.Error()}, http.StatusBadRequest)
		return
	}
	if err := validateMax("age", age, 128); err != nil {
		sendJsonResponse(w, Response{"error": err.Error()}, http.StatusBadRequest)
		return
	}
	//	log.Println("age ok")
	ctx, _ := context.WithCancel(r.Context())
	in := CreateParams{
		Login:  login,
		Name:   name,
		Status: status,
		Age:    age,
	}
	user, err := srv.Create(ctx, in)

	if err != nil {
		var stHTTP int
		switch ar := err.(type) {
		case ApiError:
			stHTTP = ar.HTTPStatus
		default:
			stHTTP = http.StatusInternalServerError
		}
		sendJsonResponse(w, Response{"error": err.Error()}, stHTTP)
		return
	}
	sendJsonResponse(w, Response{"error": "", "response": user}, http.StatusOK)
}

func (srv *MyApi) handlerProfile(w http.ResponseWriter, r *http.Request) {
	login := r.FormValue("login")
	if err := validateRequired("login", login); err != nil {
		sendJsonResponse(w, Response{"error": err.Error()}, http.StatusBadRequest)
		return
	}

	ctx, _ := context.WithCancel(r.Context())
	in := ProfileParams{
		Login: login,
	}
	user, err := srv.Profile(ctx, in)

	if err != nil {
		var stHTTP int
		switch ar := err.(type) {
		case ApiError:
			stHTTP = ar.HTTPStatus
		default:
			stHTTP = http.StatusInternalServerError
		}
		sendJsonResponse(w, Response{"error": err.Error()}, stHTTP)
		return
	}
	sendJsonResponse(w, Response{"error": "", "response": user}, http.StatusOK)
}

func (srv *OtherApi) handlerCreate(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("X-Auth") != "100500" {
		sendJsonResponse(w, Response{"error": "unauthorized"}, http.StatusForbidden)
		return
	}
	username := r.FormValue("username")
	if err := validateRequired("username", username); err != nil {
		sendJsonResponse(w, Response{"error": err.Error()}, http.StatusBadRequest)
		return
	}
	if err := validateMin("username", username, 3); err != nil {
		sendJsonResponse(w, Response{"error": err.Error()}, http.StatusBadRequest)
		return
	}
	name := r.FormValue("account_name")

	lClass := map[string]struct{}{
		"warrior":  {},
		"sorcerer": {},
		"rouge":    {},
	}
	class := r.FormValue("class")
	if class == "" {
		class = "warrior"
	}
	_, ok := lClass[class]
	if !ok {
		sendJsonResponse(w, Response{"error": "class must be one of [warrior, sorcerer, rouge]"}, http.StatusBadRequest)
		return
	}

	level, err := strconv.Atoi(r.FormValue("level"))
	if err != nil {
		sendJsonResponse(w, Response{"error": "age must be int"}, http.StatusBadRequest)
		return
	}
	if err := validateMin("level", level, 1); err != nil {
		sendJsonResponse(w, Response{"error": err.Error()}, http.StatusBadRequest)
		return
	}
	if err := validateMax("level", level, 50); err != nil {
		sendJsonResponse(w, Response{"error": err.Error()}, http.StatusBadRequest)
		return
	}

	ctx, _ := context.WithCancel(r.Context())
	in := OtherCreateParams{
		Username: username,
		Name:     name,
		Class:    class,
		Level:    level,
	}
	user, err := srv.Create(ctx, in)

	if err != nil {
		var stHTTP int
		switch ar := err.(type) {
		case ApiError:
			stHTTP = ar.HTTPStatus
		default:
			stHTTP = http.StatusInternalServerError
		}
		sendJsonResponse(w, Response{"error": err.Error()}, stHTTP)
		return
	}
	sendJsonResponse(w, Response{"error": "", "response": user}, http.StatusOK)
}

func (srv *MyApi) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.URL.Path == "/user/profile":
		srv.handlerProfile(w, r)
	case r.URL.Path == "/user/create":
		if r.Method == http.MethodPost {
			//			fmt.Println("metod post")
			srv.handlerCreate(w, r)
		} else {
			sendJsonResponse(w, Response{"error": "bad method"}, http.StatusNotAcceptable)
		}
	default:
		//		fmt.Println("metod unknow")
		sendJsonResponse(w, Response{"error": "unknown method"}, http.StatusNotFound)

	}
}

func (srv *OtherApi) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.URL.Path == "/user/create":
		if r.Method == http.MethodPost {
			//			fmt.Println("metod post")
			srv.handlerCreate(w, r)
		} else {
			sendJsonResponse(w, Response{"error": "bad method"}, http.StatusNotAcceptable)
		}
	default:
		sendJsonResponse(w, Response{"error": "unknown method"}, http.StatusNotFound)
	}
}

package main

import (
	"fmt"
	"net/http"
)

func (app *application) logError(r *http.Request, err error) {
	app.logger.PrintErr(err, map[string]string{
		"request_method": r.Method,
		"request_url":    r.URL.String(),
	})
}

func (app *application) errResponse(w http.ResponseWriter, r *http.Request, status int, msg interface{}) {
	data := envelope{"error": msg}
	err := app.writeJSON(w, status, data, nil)
	if err != nil {
		app.logError(r, err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (app *application) serverErrResponse(w http.ResponseWriter,
	r *http.Request, err error) {
	app.logError(r, err)
	msg := "the server encountered a problem and could not process your request"
	app.errResponse(w, r, http.StatusInternalServerError, msg)
}

func (app *application) notFountResponse(w http.ResponseWriter, r *http.Request) {
	msg := "the requested resource could not be found"
	app.errResponse(w, r, http.StatusNotFound, msg)
}

func (app *application) methodNotAllowResponse(w http.ResponseWriter, r *http.Request) {
	msg := fmt.Sprintf("the %s mehtod is not supported for this resource", r.Method)
	app.errResponse(w, r, http.StatusMethodNotAllowed, msg)
}

func (app *application) badRequestResponse(w http.ResponseWriter, r *http.Request, err error) {
	app.errResponse(w, r, http.StatusBadRequest, err.Error())
}

func (app *application) failedValidationResponse(w http.ResponseWriter, r *http.Request, errs map[string]string) {
	app.errResponse(w, r, http.StatusUnprocessableEntity, errs)
}

func (app *application) editConflictResponse(w http.ResponseWriter, r *http.Request) {
	msg := "unable to update the record due to an edit conflict, please try later"
	app.errResponse(w, r, http.StatusConflict, msg)
}

func (app *application) rateLimitExceededResponse(w http.ResponseWriter, r *http.Request) {
	msg := "rate limit exceeded"
	app.errResponse(w, r, http.StatusTooManyRequests, msg)
}

func (app *application) invalidCredentialsResponse(w http.ResponseWriter, r *http.Request) {
	msg := "invalid authentication credentials"
	app.errResponse(w, r, http.StatusTooManyRequests, msg)
}

func (app *application) invalidAuthenticationTokenResponse(w http.ResponseWriter, r *http.Request) {
	msg := "invalid or missing authentication token"
	app.errResponse(w, r, http.StatusTooManyRequests, msg)
}

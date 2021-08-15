package main

import (
	"errors"
	"net/http"
	"time"

	"github.com/datewu/toushi"
	"github.com/datewu/xyz/internal/data"
	"github.com/datewu/xyz/internal/validator"
)

func createAuthenticationTokenHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	err := toushi.ReadJSON(w, r, &input)
	if err != nil {
		toushi.BadRequestResponse(err).ServeHTTP(w, r)
		return
	}

	v := validator.New()
	data.ValidateEmail(v, input.Email)
	data.ValidatePasswordPlaintext(v, input.Password)

	if !v.Valid() {
		toushi.FailedValidationResponse(v.Errors).ServeHTTP(w, r)
		return
	}

	user, err := models.Users.GetByEmail(input.Email)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			toushi.InvalidCredentialsResponse.ServeHTTP(w, r)
		default:
			toushi.ServerErrResponse(err).ServeHTTP(w, r)
		}
		return
	}
	match, err := user.Password.Matches(input.Password)
	if err != nil {
		toushi.ServerErrResponse(err).ServeHTTP(w, r)
		return
	}
	if !match {
		toushi.InvalidCredentialsResponse.ServeHTTP(w, r)
		return
	}
	t, err := models.Tokens.New(user.ID, 24*time.Hour, data.ScopeAuthentication)
	if err != nil {
		toushi.ServerErrResponse(err).ServeHTTP(w, r)
		return
	}
	toushi.WriteJSON(w, http.StatusCreated, toushi.Envelope{"authentication_token": t}, nil)
}

func createPwdResetTokenHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email string `json:"email"`
	}
	err := toushi.ReadJSON(w, r, &input)
	if err != nil {
		toushi.BadRequestResponse(err).ServeHTTP(w, r)
		return
	}

	v := validator.New()
	data.ValidateEmail(v, input.Email)

	if !v.Valid() {
		toushi.FailedValidationResponse(v.Errors).ServeHTTP(w, r)
		return
	}

	user, err := models.Users.GetByEmail(input.Email)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			v.AddErr("email", "no matching email address found")
			toushi.FailedValidationResponse(v.Errors).ServeHTTP(w, r)
		default:
			toushi.ServerErrResponse(err).ServeHTTP(w, r)
		}
		return
	}
	if !user.Activated {
		v.AddErr("email", "user account must be actived")
		toushi.FailedValidationResponse(v.Errors).ServeHTTP(w, r)
		return
	}
	// t, err := models.Tokens.New(user.ID, 45*time.Minute, data.ScopePwdReset)
	_, err = models.Tokens.New(user.ID, 45*time.Minute, data.ScopePwdReset)
	if err != nil {
		toushi.ServerErrResponse(err).ServeHTTP(w, r)
		return
	}
	// app.background(func() {
	// 	data := map[string]interface{}{"passwordResetToken": t.Plaintext}
	// 	err = app.mailer.Send(user.Email, "token_password_reset.tmpl", data)
	// 	if err != nil {
	// 		app.logger.PrintErr(err, nil)
	// 	}
	// })
	container := toushi.Envelope{"message": "an email will be sent to you containing password rest instructions"}
	toushi.WriteJSON(w, http.StatusAccepted, container, nil)
}

func createActivationTokenHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email string `json:"email"`
	}
	err := toushi.ReadJSON(w, r, &input)
	if err != nil {
		toushi.BadRequestResponse(err).ServeHTTP(w, r)
		return
	}

	v := validator.New()
	data.ValidateEmail(v, input.Email)

	if !v.Valid() {
		toushi.FailedValidationResponse(v.Errors).ServeHTTP(w, r)
		return
	}

	user, err := models.Users.GetByEmail(input.Email)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			v.AddErr("email", "no matching email address found")
			toushi.FailedValidationResponse(v.Errors).ServeHTTP(w, r)
		default:
			toushi.ServerErrResponse(err).ServeHTTP(w, r)
		}
		return
	}
	if user.Activated {
		v.AddErr("email", "user has already been activated")
		toushi.FailedValidationResponse(v.Errors).ServeHTTP(w, r)
		return
	}
	// t, err := app.models.Tokens.New(user.ID, 3*24*time.Hour, data.ScopeActivation)
	_, err = models.Tokens.New(user.ID, 3*24*time.Hour, data.ScopeActivation)
	if err != nil {
		toushi.ServerErrResponse(err).ServeHTTP(w, r)
		return
	}
	// app.background(func() {
	// 	data := map[string]interface{}{"activationToken": t.Plaintext}
	// 	err = app.mailer.Send(user.Email, "token_activation.tmpl", data)
	// 	if err != nil {
	// 		app.logger.PrintErr(err, nil)
	// 	}
	// })
	container := toushi.Envelope{"message": "an email will be sent to you containing activation instructions"}
	toushi.WriteJSON(w, http.StatusAccepted, container, nil)
}

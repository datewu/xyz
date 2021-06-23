package main

import (
	"errors"
	"net/http"
	"time"

	"github.com/datewu/xyz/internal/data"
	"github.com/datewu/xyz/internal/validator"
)

func (app *application) createAuthenticationTokenHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	v := validator.New()
	data.ValidateEmail(v, input.Email)
	data.ValidatePasswordPlaintext(v, input.Password)

	if !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	user, err := app.models.Users.GetByEmail(input.Email)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.invalidCredentialsResponse(w, r)
		default:
			app.serverErrResponse(w, r, err)
		}
		return
	}
	match, err := user.Password.Matches(input.Password)
	if err != nil {
		app.serverErrResponse(w, r, err)
		return
	}
	if !match {
		app.invalidCredentialsResponse(w, r)
		return
	}
	t, err := app.models.Tokens.New(user.ID, 24*time.Hour, data.ScopeAuthentication)
	if err != nil {
		app.serverErrResponse(w, r, err)
		return
	}
	err = app.writeJSON(w, http.StatusCreated, envelope{"authentication_token": t}, nil)
	if err != nil {
		app.serverErrResponse(w, r, err)
	}
}

func (app *application) createPwdResetTokenHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email string `json:"email"`
	}
	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	v := validator.New()
	data.ValidateEmail(v, input.Email)

	if !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	user, err := app.models.Users.GetByEmail(input.Email)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			v.AddErr("email", "no matching email address found")
			app.failedValidationResponse(w, r, v.Errors)
		default:
			app.serverErrResponse(w, r, err)
		}
		return
	}
	if !user.Activated {
		v.AddErr("email", "user account must be actived")
		app.failedValidationResponse(w, r, v.Errors)
		return
	}
	t, err := app.models.Tokens.New(user.ID, 45*time.Minute, data.ScopePwdReset)
	if err != nil {
		app.serverErrResponse(w, r, err)
		return
	}
	app.background(func() {
		data := map[string]interface{}{"passwordResetToken": t.Plaintext}
		err = app.mailer.Send(user.Email, "token_password_reset.tmpl", data)
		if err != nil {
			app.logger.PrintErr(err, nil)
		}
	})
	container := envelope{"message": "an email will be sent to you containing password rest instructions"}
	err = app.writeJSON(w, http.StatusAccepted, container, nil)
	if err != nil {
		app.serverErrResponse(w, r, err)
	}
}

func (app *application) createActivationTokenHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email string `json:"email"`
	}
	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	v := validator.New()
	data.ValidateEmail(v, input.Email)

	if !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	user, err := app.models.Users.GetByEmail(input.Email)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			v.AddErr("email", "no matching email address found")
			app.failedValidationResponse(w, r, v.Errors)
		default:
			app.serverErrResponse(w, r, err)
		}
		return
	}
	if user.Activated {
		v.AddErr("email", "user has already been activated")
		app.failedValidationResponse(w, r, v.Errors)
		return
	}
	t, err := app.models.Tokens.New(user.ID, 3*24*time.Hour, data.ScopeActivation)
	if err != nil {
		app.serverErrResponse(w, r, err)
		return
	}
	app.background(func() {
		data := map[string]interface{}{"activationToken": t.Plaintext}
		err = app.mailer.Send(user.Email, "token_activation.tmpl", data)
		if err != nil {
			app.logger.PrintErr(err, nil)
		}
	})
	container := envelope{"message": "an email will be sent to you containing activation instructions"}
	err = app.writeJSON(w, http.StatusAccepted, container, nil)
	if err != nil {
		app.serverErrResponse(w, r, err)
	}
}

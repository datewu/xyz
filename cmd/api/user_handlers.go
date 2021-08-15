package main

import (
	"errors"
	"net/http"
	"time"

	"github.com/datewu/toushi"
	"github.com/datewu/xyz/internal/data"
	"github.com/datewu/xyz/internal/validator"
)

func registerUserHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	err := toushi.ReadJSON(w, r, &input)
	if err != nil {
		toushi.BadRequestResponse(err).ServeHTTP(w, r)
		return
	}

	user := &data.User{
		Name:      input.Name,
		Email:     input.Email,
		Activated: false,
	}
	err = user.Password.Set(input.Password)
	if err != nil {
		toushi.ServerErrResponse(err).ServeHTTP(w, r)
		return
	}
	v := validator.New()
	if data.ValidateUser(v, user); !v.Valid() {
		toushi.FailedValidationResponse(v.Errors).ServeHTTP(w, r)
		return
	}

	err = models.Users.Insert(user)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrDuplicateEmail):
			v.AddErr("email", "a user with this emal address already exists")
			toushi.FailedValidationResponse(v.Errors).ServeHTTP(w, r)
		default:
			toushi.ServerErrResponse(err).ServeHTTP(w, r)
		}
		return
	}
	err = models.Permissions.AddForUser(user.ID, "movies:read")
	if err != nil {
		toushi.ServerErrResponse(err).ServeHTTP(w, r)
		return
	}

	//	token, err := models.Tokens.New(user.ID, 3*24*time.Hour, data.ScopeActivation)
	_, err = models.Tokens.New(user.ID, 3*24*time.Hour, data.ScopeActivation)
	if err != nil {
		toushi.ServerErrResponse(err).ServeHTTP(w, r)
		return
	}
	// sendMail := func() {
	// 	data := map[string]interface{}{
	// 		"activationToken": token.Plaintext,
	// 		"userID":          user.ID,
	// 	}
	// 	err = app.mailer.Send(user.Email, "user_welcome.tmpl", data)
	// 	if err != nil {
	// 		app.logger.PrintErr(err, nil)
	// 	}
	// }
	// app.background(sendMail)
	toushi.WriteJSON(w, http.StatusAccepted, toushi.Envelope{"user": user}, nil)
}

func activateUserHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		TokenPlaintext string `json:"token"`
	}
	err := toushi.ReadJSON(w, r, &input)
	if err != nil {
		toushi.BadRequestResponse(err).ServeHTTP(w, r)
		return
	}
	v := validator.New()
	if data.ValidateTokenPlaintext(v, input.TokenPlaintext); !v.Valid() {
		toushi.FailedValidationResponse(v.Errors).ServeHTTP(w, r)
		return
	}
	user, err := models.Users.GetForToken(data.ScopeActivation, input.TokenPlaintext)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			v.AddErr("token", "invalid or expired activation token")
			toushi.FailedValidationResponse(v.Errors).ServeHTTP(w, r)
		default:
			toushi.ServerErrResponse(err).ServeHTTP(w, r)
		}
		return
	}
	user.Activated = true
	err = models.Users.Update(user)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrEditConflict):
			toushi.EditConflictResponse.ServeHTTP(w, r)
		default:
			toushi.ServerErrResponse(err).ServeHTTP(w, r)
		}
		return
	}
	err = models.Tokens.DeleteAllForUser(data.ScopeActivation, user.ID)
	if err != nil {
		toushi.ServerErrResponse(err).ServeHTTP(w, r)
		return
	}
	toushi.WriteJSON(w, http.StatusOK, toushi.Envelope{"user": user}, nil)
}

func updateUserPasswordHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Password       string `json:"password"`
		TokenPlaintext string `json:"token"`
	}
	err := toushi.ReadJSON(w, r, &input)
	if err != nil {
		toushi.BadRequestResponse(err).ServeHTTP(w, r)
		return
	}
	v := validator.New()
	data.ValidatePasswordPlaintext(v, input.Password)
	data.ValidateTokenPlaintext(v, input.TokenPlaintext)
	if !v.Valid() {
		toushi.FailedValidationResponse(v.Errors).ServeHTTP(w, r)
		return
	}
	user, err := models.Users.GetForToken(data.ScopePwdReset, input.TokenPlaintext)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			v.AddErr("token", "invalid or expired password reset token")
			toushi.FailedValidationResponse(v.Errors).ServeHTTP(w, r)
		default:
			toushi.ServerErrResponse(err).ServeHTTP(w, r)
		}
		return
	}
	err = user.Password.Set(input.Password)
	if err != nil {
		toushi.ServerErrResponse(err).ServeHTTP(w, r)
		return
	}
	err = models.Users.Update(user)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrEditConflict):
			toushi.EditConflictResponse.ServeHTTP(w, r)
		default:
			toushi.ServerErrResponse(err).ServeHTTP(w, r)
		}
		return
	}
	err = models.Tokens.DeleteAllForUser(data.ScopePwdReset, user.ID)
	if err != nil {
		toushi.ServerErrResponse(err).ServeHTTP(w, r)
		return
	}
	toushi.WriteJSON(w, http.StatusOK, toushi.Envelope{"message": "your password was successfully reset"}, nil)
}

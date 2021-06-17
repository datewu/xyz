package main

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/datewu/xyz/internal/data"
	"github.com/datewu/xyz/internal/validator"
)

func (app *application) createMovieHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Title   string       `json:"title"`
		Year    int32        `json:"year"`
		Runtime data.Runtime `json:"runtime"`
		Genres  []string     `json:"genres"`
	}
	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}
	m := &data.Movie{
		Title:   input.Title,
		Year:    input.Year,
		Runtime: input.Runtime,
		Genres:  input.Genres,
	}

	v := validator.New()
	if data.ValidateMovie(v, m); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	err = app.models.Movies.Insert(m)
	if err != nil {
		app.serverErrResponse(w, r, err)
		return
	}
	hs := make(http.Header)
	hs.Set("Location", fmt.Sprintf("/v1/movies/%d", m.ID))
	err = app.writeJSON(w, http.StatusCreated, envelope{"movie": m}, hs)
	if err != nil {
		app.serverErrResponse(w, r, err)
		return
	}
}

func (app *application) showMovieHandler(w http.ResponseWriter, r *http.Request) {
	id, err := app.readIDParam(r)
	if err != nil {
		app.notFountResponse(w, r)
		return
	}
	m, err := app.models.Movies.Get(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.notFountResponse(w, r)
		default:
			app.serverErrResponse(w, r, err)
		}
		return
	}
	err = app.writeJSON(w, http.StatusOK, envelope{"movie": m}, nil)
	if err != nil {
		app.serverErrResponse(w, r, err)
	}
}

func (app *application) updateMovieHandler(w http.ResponseWriter, r *http.Request) {
	id, err := app.readIDParam(r)
	if err != nil {
		app.notFountResponse(w, r)
		return
	}
	m, err := app.models.Movies.Get(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.notFountResponse(w, r)
		default:
			app.serverErrResponse(w, r, err)
		}
		return
	}
	var input struct {
		Title   string       `json:"title"`
		Year    int32        `json:"year"`
		Runtime data.Runtime `json:"runtime"`
		Genres  []string     `json:"genres"`
	}
	err = app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}
	m.Title = input.Title
	m.Year = input.Year
	m.Runtime = input.Runtime
	m.Genres = input.Genres

	v := validator.New()
	if data.ValidateMovie(v, m); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}
	err = app.models.Movies.Update(m)
	if err != nil {
		app.serverErrResponse(w, r, err)
	}
	err = app.writeJSON(w, http.StatusOK, envelope{"movie": m}, nil)
	if err != nil {
		app.serverErrResponse(w, r, err)
	}
}
func (app *application) deleteMovieHandler(w http.ResponseWriter, r *http.Request) {
	id, err := app.readIDParam(r)
	if err != nil {
		app.notFountResponse(w, r)
		return
	}
	err = app.models.Movies.Delete(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.notFountResponse(w, r)
		default:
			app.serverErrResponse(w, r, err)
		}
		return
	}
	err = app.writeJSON(w, http.StatusOK, envelope{"message": "movie successfully deleted"}, nil)
	if err != nil {
		app.serverErrResponse(w, r, err)
	}
}

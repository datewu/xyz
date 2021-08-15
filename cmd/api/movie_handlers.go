package main

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/datewu/toushi"
	"github.com/datewu/xyz/internal/data"
	"github.com/datewu/xyz/internal/validator"
)

func createMovieHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Title   string       `json:"title"`
		Year    int32        `json:"year"`
		Runtime data.Runtime `json:"runtime"`
		Genres  []string     `json:"genres"`
	}
	err := toushi.ReadJSON(w, r, &input)
	if err != nil {
		toushi.BadRequestResponse(err).ServeHTTP(w, r)
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
		toushi.FailedValidationResponse(v.Errors).ServeHTTP(w, r)
		return
	}

	err = models.Movies.Insert(m)
	if err != nil {
		toushi.ServerErrResponse(err).ServeHTTP(w, r)
		return
	}
	hs := make(http.Header)
	hs.Set("Location", fmt.Sprintf("/v1/movies/%d", m.ID))
	toushi.WriteJSON(w, http.StatusCreated, toushi.Envelope{"movie": m}, hs)
}

func showMovieHandler(w http.ResponseWriter, r *http.Request) {
	id, err := toushi.ReadIDParam(r)
	if err != nil {
		toushi.NotFountResponse.ServeHTTP(w, r)
		return
	}
	m, err := models.Movies.Get(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			toushi.NotFountResponse.ServeHTTP(w, r)
		default:
			toushi.ServerErrResponse(err).ServeHTTP(w, r)
		}
		return
	}
	toushi.WriteJSON(w, http.StatusOK, toushi.Envelope{"movie": m}, nil)
}

func updateMovieHandler(w http.ResponseWriter, r *http.Request) {
	id, err := toushi.ReadIDParam(r)
	if err != nil {
		toushi.NotFountResponse.ServeHTTP(w, r)
		return
	}
	m, err := models.Movies.Get(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			toushi.NotFountResponse.ServeHTTP(w, r)
		default:
			toushi.ServerErrResponse(err).ServeHTTP(w, r)
		}
		return
	}

	if cliVer := r.Header.Get("X-Expected-Version"); cliVer != "" {
		if strconv.FormatInt(int64(m.Version), 32) != cliVer {
			toushi.EditConflictResponse.ServeHTTP(w, r)
			return
		}
	}
	var input struct {
		Title   *string       `json:"title"`
		Year    *int32        `json:"year"`
		Runtime *data.Runtime `json:"runtime"`
		Genres  []string      `json:"genres"`
	}
	err = toushi.ReadJSON(w, r, &input)
	if err != nil {
		toushi.BadRequestResponse(err).ServeHTTP(w, r)
		return
	}
	if input.Title != nil {
		m.Title = *input.Title
	}
	if input.Year != nil {
		m.Year = *input.Year
	}
	if input.Runtime != nil {
		m.Runtime = *input.Runtime
	}
	if input.Genres != nil {
		m.Genres = input.Genres
	}

	v := validator.New()
	if data.ValidateMovie(v, m); !v.Valid() {
		toushi.FailedValidationResponse(v.Errors).ServeHTTP(w, r)
		return
	}
	err = models.Movies.Update(m)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrEditConflict):
			toushi.EditConflictResponse.ServeHTTP(w, r)
		default:
			toushi.ServerErrResponse(err).ServeHTTP(w, r)
		}
		return
	}
	toushi.WriteJSON(w, http.StatusOK, toushi.Envelope{"movie": m}, nil)
}

func deleteMovieHandler(w http.ResponseWriter, r *http.Request) {
	id, err := toushi.ReadIDParam(r)
	if err != nil {
		toushi.NotFountResponse.ServeHTTP(w, r)
		return
	}
	err = models.Movies.Delete(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			toushi.NotFountResponse.ServeHTTP(w, r)
		default:
			toushi.ServerErrResponse(err).ServeHTTP(w, r)
		}
		return
	}
	toushi.WriteJSON(w, http.StatusOK, toushi.Envelope{"message": "movie successfully deleted"}, nil)
}

func listMovieHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Title  string
		Genres []string
		data.Filters
	}

	v := validator.New()
	qs := r.URL.Query()
	input.Title = toushi.ReadString(qs, "title", "")
	input.Genres = toushi.ReadCSV(qs, "genres", []string{})
	input.Page = toushi.ReadInt(qs, "page", 1)
	input.PageSize = toushi.ReadInt(qs, "page_size", 20)
	input.Sort = toushi.ReadString(qs, "sort", "id")
	input.SortSafelist = []string{"id", "title", "year", "runtime",
		"-id", "-title", "-year", "-runtime"}
	if data.ValidateFilters(v, input.Filters); !v.Valid() {
		toushi.FailedValidationResponse(v.Errors).ServeHTTP(w, r)
		return
	}
	movies, metadata, err := models.Movies.GetAll(input.Title, input.Genres, input.Filters)
	if err != nil {
		toushi.ServerErrResponse(err).ServeHTTP(w, r)
		return
	}

	toushi.WriteJSON(w, http.StatusOK, toushi.Envelope{"movies": movies, "metadata": metadata}, nil)
}

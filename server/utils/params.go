package utils

import (
	"net/http"
	"strconv"
)

const (
	PageDefault    = 0
	PerPageDefault = 60
	PerPageMaximum = 200
)

func GetPageAndPerPage(r *http.Request) (page, perPage int) {
	query := r.URL.Query()
	if val, err := strconv.Atoi(query.Get("page")); err != nil || val < 0 {
		page = PageDefault
	} else {
		page = val
	}

	val, err := strconv.Atoi(query.Get("per_page"))
	switch {
	case err != nil || val < 0:
		perPage = PerPageDefault
	case val > PerPageMaximum:
		perPage = PerPageMaximum
	default:
		perPage = val
	}

	return page, perPage
}

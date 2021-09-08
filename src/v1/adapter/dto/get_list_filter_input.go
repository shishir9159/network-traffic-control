package dto

import (
	"net/url"
	"strconv"
)

type GetListFilterInput struct {
	FetchMode       string            `json:"fetchMode"`
	FilterParams    string            `json:"filterParams"`
	FilterParamsMap map[string]string `json:"filterParamsMap"`
}

type SearchFilterAndPaginationInput struct {
	Search       string            `json:"search"`
	FilterParams map[string]string `json:"filterParams"`
	Page         int               `json:"page"`
	Pagelen      int               `default:"20" json:"pagelen"`
	Sort         string            `json:"sort"`
	Desc         bool              `json:"desc"`
}

func GetSearchFilterAndPaginationInput(values url.Values) SearchFilterAndPaginationInput {
	input := new(SearchFilterAndPaginationInput)
	input = &SearchFilterAndPaginationInput{Page: -1, Pagelen: 20, FilterParams: make(map[string]string)}
	for key, _ := range values {
		if key == "search" {
			input.Search = values.Get("search")
		} else if key == "sort" {
			input.Sort = values.Get("sort")
		} else if key == "page" {
			if pg, err := strconv.Atoi(values.Get("page")); err == nil && pg != 0 {
				input.Page = pg
			}
		} else if key == "pagelen" {
			if pglen, err := strconv.Atoi(values.Get("pagelen")); err == nil && pglen > 0 {
				input.Pagelen = pglen
			}
		} else if key == "desc" {
			if dsc, err := strconv.ParseBool(values.Get("desc")); err == nil {
				input.Desc = dsc
			}
		} else {
			vale := values.Get(key)
			input.FilterParams[key] = vale
		}
	}
	return *input
}

package routers

import (
	"testing"

	"github.com/gorilla/mux"
	corev2 "github.com/sensu/core/v2"
	"github.com/sensu/sensu-go/testing/mockstore"
)

func TestPipelinesRouter(t *testing.T) {
	s := &mockstore.V2MockStore{}
	cs := new(mockstore.ConfigStore)
	s.On("GetConfigStore").Return(cs)
	router := NewPipelinesRouter(s)
	parentRouter := mux.NewRouter().PathPrefix(corev2.URLPrefix).Subrouter()
	router.Mount(parentRouter)

	empty := &corev2.Pipeline{}
	fixture := corev2.FixturePipeline("foo", "default")

	tests := []routerTestCase{}
	tests = append(tests, getTestCases[*corev2.Pipeline](fixture)...)
	tests = append(tests, listTestCases[*corev2.Pipeline](empty)...)
	tests = append(tests, createTestCases(fixture)...)
	tests = append(tests, updateTestCases(fixture)...)
	tests = append(tests, deleteTestCases(fixture)...)
	for _, tt := range tests {
		run(t, tt, parentRouter, s)
	}
}

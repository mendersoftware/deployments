// Copyright 2023 Northern.tech AS
//
//    Licensed under the Apache License, Version 2.0 (the "License");
//    you may not use this file except in compliance with the License.
//    You may obtain a copy of the License at
//
//        http://www.apache.org/licenses/LICENSE-2.0
//
//    Unless required by applicable law or agreed to in writing, software
//    distributed under the License is distributed on an "AS IS" BASIS,
//    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//    See the License for the specific language governing permissions and
//    limitations under the License.

package http

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/ant0ine/go-json-rest/rest"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/mendersoftware/deployments/app"
	mapp "github.com/mendersoftware/deployments/app/mocks"
	"github.com/mendersoftware/deployments/model"
	"github.com/mendersoftware/deployments/utils/restutil/view"
)

func TestPutReleaseTags(t *testing.T) {
	t.Parallel()

	type testCase struct {
		Name string

		App func(t *testing.T, self *testCase) *mapp.App
		*http.Request

		StatusCode int
	}

	testCases := []testCase{{
		Name: "ok",

		Request: func() *http.Request {
			b, _ := json.Marshal(model.Tags{"one", "one", "two", "three"})

			req, _ := http.NewRequest(
				http.MethodPut,
				fmt.Sprintf("http://localhost:1234%s",
					strings.ReplaceAll(ApiUrlManagementV2ReleaseTags, "#name", "release-mc-release-face")),
				bytes.NewReader(b),
			)
			return req
		}(),

		App: func(t *testing.T, self *testCase) *mapp.App {
			appie := new(mapp.App)
			expectedTags := model.Tags{"one", "two", "three"}
			appie.On("ReplaceReleaseTags",
				contextMatcher(),
				"release-mc-release-face",
				expectedTags).
				Return(nil)
			return appie
		},

		StatusCode: http.StatusNoContent,
	}, {
		Name: "error/internal",

		Request: func() *http.Request {
			b, _ := json.Marshal(model.Tags{"one", "two", "three"})

			req, _ := http.NewRequest(
				http.MethodPut,
				fmt.Sprintf("http://localhost:1234%s",
					strings.ReplaceAll(ApiUrlManagementV2ReleaseTags, "#name", "release-mc-release-face")),
				bytes.NewReader(b),
			)
			return req
		}(),

		App: func(t *testing.T, self *testCase) *mapp.App {
			appie := new(mapp.App)
			expectedTags := model.Tags{"one", "two", "three"}
			appie.On("ReplaceReleaseTags",
				contextMatcher(),
				"release-mc-release-face",
				expectedTags).
				Return(errors.New("internal error"))
			return appie
		},

		StatusCode: http.StatusInternalServerError,
	}, {
		Name: "error/too many unique tags",

		Request: func() *http.Request {
			b, _ := json.Marshal(model.Tags{"one", "two", "three"})

			req, _ := http.NewRequest(
				http.MethodPut,
				fmt.Sprintf("http://localhost:1234%s",
					strings.ReplaceAll(ApiUrlManagementV2ReleaseTags, "#name", "release-mc-release-face")),
				bytes.NewReader(b),
			)
			return req
		}(),

		App: func(t *testing.T, self *testCase) *mapp.App {
			appie := new(mapp.App)
			expectedTags := model.Tags{"one", "two", "three"}
			appie.On("ReplaceReleaseTags",
				contextMatcher(),
				"release-mc-release-face",
				expectedTags).
				Return(model.ErrTooManyUniqueTags)
			return appie
		},

		StatusCode: http.StatusConflict,
	}, {
		Name: "error/release not found",

		Request: func() *http.Request {
			b, _ := json.Marshal(model.Tags{"one", "two", "three"})

			req, _ := http.NewRequest(
				http.MethodPut,
				fmt.Sprintf("http://localhost:1234%s",
					strings.ReplaceAll(ApiUrlManagementV2ReleaseTags, "#name", "release-mc-release-face")),
				bytes.NewReader(b),
			)
			return req
		}(),

		App: func(t *testing.T, self *testCase) *mapp.App {
			appie := new(mapp.App)
			expectedTags := model.Tags{"one", "two", "three"}
			appie.On("ReplaceReleaseTags",
				contextMatcher(),
				"release-mc-release-face",
				expectedTags).
				Return(app.ErrReleaseNotFound)
			return appie
		},

		StatusCode: http.StatusNotFound,
	}, {
		Name: "error/too many tags",

		Request: func() *http.Request {
			tags := make(model.Tags, model.TagsMaxPerRelease+1)
			for i := range tags {
				tags[i] = model.Tag("field" + strconv.Itoa(i))
			}
			b, _ := json.Marshal(tags)

			req, _ := http.NewRequest(
				http.MethodPut,
				fmt.Sprintf("http://localhost:1234%s",
					strings.ReplaceAll(ApiUrlManagementV2ReleaseTags, "#name", "release-mc-release-face")),
				bytes.NewReader(b),
			)
			return req
		}(),

		App: func(t *testing.T, self *testCase) *mapp.App {
			return new(mapp.App)
		},

		StatusCode: http.StatusBadRequest,
	}, {
		Name: "ok/many duplicate tags",

		Request: func() *http.Request {
			tags := make(model.Tags, model.TagsMaxPerRelease+1)
			for i := range tags {
				tags[i] = model.Tag("field")
			}
			b, _ := json.Marshal(tags)

			req, _ := http.NewRequest(
				http.MethodPut,
				fmt.Sprintf("http://localhost:1234%s",
					strings.ReplaceAll(ApiUrlManagementV2ReleaseTags, "#name", "release-mc-release-face")),
				bytes.NewReader(b),
			)
			return req
		}(),

		App: func(t *testing.T, self *testCase) *mapp.App {
			appie := new(mapp.App)
			expectedTags := model.Tags{"field"}
			appie.On("ReplaceReleaseTags",
				contextMatcher(),
				"release-mc-release-face",
				expectedTags).
				Return(nil)
			return appie
		},

		StatusCode: http.StatusNoContent,
	}, {
		Name: "error/malformed JSON",

		Request: func() *http.Request {
			req, _ := http.NewRequest(
				http.MethodPut,
				fmt.Sprintf("http://localhost:1234%s",
					strings.ReplaceAll(ApiUrlManagementV2ReleaseTags, "#name", "release-mc-release-face")),
				bytes.NewReader([]byte("not json")),
			)
			return req
		}(),

		App: func(t *testing.T, self *testCase) *mapp.App {
			return new(mapp.App)
		},

		StatusCode: http.StatusBadRequest,
	}, {
		Name: "error/empty release name",

		Request: func() *http.Request {
			req, _ := http.NewRequest(
				http.MethodPut,
				fmt.Sprintf("http://localhost:1234%s",
					strings.ReplaceAll(ApiUrlManagementV2ReleaseTags, "#name", "")),
				bytes.NewReader([]byte("[]")),
			)
			return req
		}(),

		App: func(t *testing.T, self *testCase) *mapp.App {
			return new(mapp.App)
		},

		StatusCode: http.StatusNotFound,
	}}

	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.Name, func(t *testing.T) {
			appie := tc.App(t, &tc)
			defer appie.AssertExpectations(t)

			handlers := NewDeploymentsApiHandlers(nil, &view.RESTView{}, appie)
			routes := ReleasesRoutes(handlers)
			router, _ := rest.MakeRouter(routes...)
			api := rest.NewApi()
			api.SetApp(router)
			handler := api.MakeHandler()
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, tc.Request)

			rsp := w.Result()
			assert.Equal(t, tc.StatusCode, rsp.StatusCode,
				"unexpected status code from request")
		})
	}
}

func TestListReleaseTags(t *testing.T) {
	t.Parallel()

	type testCase struct {
		Name string

		App func(t *testing.T, self *testCase) *mapp.App
		*http.Request

		StatusCode int
		Tags       model.Tags
	}

	testCases := []testCase{{
		Name: "ok",

		Request: func() *http.Request {
			req, _ := http.NewRequest(
				http.MethodGet,
				fmt.Sprintf("http://localhost:1234%s",
					strings.ReplaceAll(ApiUrlManagementV2ReleaseAllTags,
						"#name", "release-mc-release-face"),
				),
				nil,
			)
			return req
		}(),

		App: func(t *testing.T, self *testCase) *mapp.App {
			appie := new(mapp.App)
			appie.On("ListReleaseTags",
				contextMatcher()).
				Return(self.Tags, nil)
			return appie
		},

		StatusCode: http.StatusOK,
		Tags:       model.Tags{"bar", "baz", "foo"},
	}, {
		Name: "error/internal",

		Request: func() *http.Request {
			req, _ := http.NewRequest(
				http.MethodGet,
				fmt.Sprintf("http://localhost:1234%s",
					strings.ReplaceAll(ApiUrlManagementV2ReleaseAllTags,
						"#name", "release-mc-release-face"),
				),
				nil,
			)
			return req
		}(),

		App: func(t *testing.T, self *testCase) *mapp.App {
			appie := new(mapp.App)
			appie.On("ListReleaseTags",
				contextMatcher()).
				Return(nil, errors.New("internal error"))
			return appie
		},

		StatusCode: http.StatusInternalServerError,
	}, {
		Name: "error/internal",

		Request: func() *http.Request {
			req, _ := http.NewRequest(
				http.MethodGet,
				fmt.Sprintf("http://localhost:1234%s",
					strings.ReplaceAll(ApiUrlManagementV2ReleaseAllTags,
						"#name", "release-mc-release-face"),
				),
				nil,
			)
			return req
		}(),

		App: func(t *testing.T, self *testCase) *mapp.App {
			appie := new(mapp.App)
			appie.On("ListReleaseTags",
				contextMatcher()).
				Return(nil, errors.New("internal error"))
			return appie
		},

		StatusCode: http.StatusInternalServerError,
	}}

	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.Name, func(t *testing.T) {
			appie := tc.App(t, &tc)
			defer appie.AssertExpectations(t)

			handlers := NewDeploymentsApiHandlers(nil, &view.RESTView{}, appie)
			routes := ReleasesRoutes(handlers)
			router, _ := rest.MakeRouter(routes...)
			api := rest.NewApi()
			api.SetApp(router)
			handler := api.MakeHandler()
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, tc.Request)

			rsp := w.Result()
			assert.Equal(t, tc.StatusCode, rsp.StatusCode,
				"unexpected status code from request")
			if tc.Tags != nil {
				var actual model.Tags
				err := json.Unmarshal(w.Body.Bytes(), &actual)
				if assert.NoError(t, err, "unexpected request body") {
					assert.Equal(t, tc.Tags, actual)
				}
			}
		})
	}
}

func TestGetReleasesUpdateTypes(t *testing.T) {
	t.Parallel()

	type testCase struct {
		Name string

		App func(t *testing.T, self *testCase) *mapp.App
		*http.Request

		StatusCode int
		Types      []string
	}

	testCases := []testCase{
		{
			Name: "ok",

			Request: func() *http.Request {
				req, _ := http.NewRequest(
					http.MethodGet,
					fmt.Sprintf("http://localhost:1234%s",
						ApiUrlManagementV2ReleaseAllUpdateTypes,
					),
					nil,
				)
				return req
			}(),

			App: func(t *testing.T, self *testCase) *mapp.App {
				appie := new(mapp.App)
				appie.On("GetReleasesUpdateTypes",
					contextMatcher()).
					Return(self.Types, nil)
				return appie
			},

			StatusCode: http.StatusOK,
			Types:      []string{"bar", "baz", "foo"},
		},
		{
			Name: "error/internal",

			Request: func() *http.Request {
				req, _ := http.NewRequest(
					http.MethodGet,
					fmt.Sprintf("http://localhost:1234%s",
						ApiUrlManagementV2ReleaseAllUpdateTypes,
					),
					nil,
				)
				return req
			}(),

			App: func(t *testing.T, self *testCase) *mapp.App {
				appie := new(mapp.App)
				appie.On("GetReleasesUpdateTypes",
					contextMatcher()).
					Return([]string{}, errors.New("internal"))
				return appie
			},

			StatusCode: http.StatusInternalServerError,
		},
	}

	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.Name, func(t *testing.T) {
			appie := tc.App(t, &tc)
			defer appie.AssertExpectations(t)

			handlers := NewDeploymentsApiHandlers(nil, &view.RESTView{}, appie)
			routes := ReleasesRoutes(handlers)
			router, _ := rest.MakeRouter(routes...)
			api := rest.NewApi()
			api.SetApp(router)
			handler := api.MakeHandler()
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, tc.Request)

			rsp := w.Result()
			assert.Equal(t, tc.StatusCode, rsp.StatusCode,
				"unexpected status code from request")
			if tc.Types != nil {
				var actual []string
				err := json.Unmarshal(w.Body.Bytes(), &actual)
				if assert.NoError(t, err, "unexpected request body") {
					assert.Equal(t, tc.Types, actual)
				}
			}
		})
	}
}

func TestPatchRelease(t *testing.T) {
	t.Parallel()

	longReleaseNotes := make([]byte, model.NotesLengthMaximumCharacters+1)
	for i := range longReleaseNotes {
		longReleaseNotes[i] = '1'
	}

	type testCase struct {
		Name string

		App func(t *testing.T, self *testCase) *mapp.App
		*http.Request

		StatusCode int
	}

	testCases := []testCase{
		{
			Name: "ok",

			Request: func() *http.Request {
				data, _ := json.Marshal(model.ReleasePatch{Notes: "New Release and fixes 2023"})
				req, _ := http.NewRequest(
					http.MethodPatch,
					fmt.Sprintf("http://localhost:1234%s",
						strings.ReplaceAll(ApiUrlManagementV2ReleasesName,
							"#name", "release-mc-release-face"),
					),
					bytes.NewReader(data),
				)
				return req
			}(),

			App: func(t *testing.T, self *testCase) *mapp.App {
				appie := new(mapp.App)
				appie.On("UpdateRelease",
					contextMatcher(),
					mock.AnythingOfType("string"),
					mock.AnythingOfType("model.ReleasePatch"),
				).Return(nil)
				return appie
			},

			StatusCode: http.StatusNoContent,
		},
		{
			Name: "error/notes too long",

			Request: func() *http.Request {
				data, _ := json.Marshal(model.ReleasePatch{Notes: model.Notes(longReleaseNotes)})
				req, _ := http.NewRequest(
					http.MethodPatch,
					fmt.Sprintf("http://localhost:1234%s",
						strings.ReplaceAll(ApiUrlManagementV2ReleasesName,
							"#name", "release-mc-release-face"),
					),
					bytes.NewReader(data),
				)
				return req
			}(),

			App: func(t *testing.T, self *testCase) *mapp.App {
				appie := new(mapp.App)
				return appie
			},

			StatusCode: http.StatusBadRequest,
		},
		{
			Name: "error/internal",

			Request: func() *http.Request {
				data, _ := json.Marshal(model.ReleasePatch{Notes: "New Release and fixes 2023"})
				req, _ := http.NewRequest(
					http.MethodPatch,
					fmt.Sprintf("http://localhost:1234%s",
						strings.ReplaceAll(ApiUrlManagementV2ReleasesName,
							"#name", "release-mc-release-face"),
					),
					bytes.NewReader(data),
				)
				return req
			}(),

			App: func(t *testing.T, self *testCase) *mapp.App {
				appie := new(mapp.App)
				appie.On("UpdateRelease",
					contextMatcher(),
					mock.AnythingOfType("string"),
					mock.AnythingOfType("model.ReleasePatch"),
				).Return(errors.New("internal error"))
				return appie
			},

			StatusCode: http.StatusInternalServerError,
		},
	}

	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.Name, func(t *testing.T) {
			appie := tc.App(t, &tc)
			defer appie.AssertExpectations(t)

			handlers := NewDeploymentsApiHandlers(nil, &view.RESTView{}, appie)
			routes := ReleasesRoutes(handlers)
			router, _ := rest.MakeRouter(routes...)
			api := rest.NewApi()
			api.SetApp(router)
			handler := api.MakeHandler()
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, tc.Request)

			rsp := w.Result()
			assert.Equal(t, tc.StatusCode, rsp.StatusCode,
				"unexpected status code from request")
		})
	}
}

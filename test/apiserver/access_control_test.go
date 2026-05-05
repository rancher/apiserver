// nolint: errcheck
package apiserver_test

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rancher/apiserver/pkg/apierror"
	"github.com/rancher/apiserver/pkg/server"
	"github.com/rancher/apiserver/pkg/types"
	"github.com/rancher/wrangler/v3/pkg/schemas/validation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// resourcePerm holds the read/write access bits for one resource type.
type resourcePerm struct {
	read  bool
	write bool
}

// userAccessControl is a test AccessControl implementation that reads the
// username from the X-User request header and enforces per-user, per-resource
// permissions stored in the perms map.
type userAccessControl struct {
	// perms maps username → schemaID → permissions.
	perms map[string]map[string]resourcePerm
}

func (u *userAccessControl) getUser(apiOp *types.APIRequest) string {
	return apiOp.Request.Header.Get("X-User")
}

func (u *userAccessControl) canRead(apiOp *types.APIRequest, schema *types.APISchema) bool {
	user := u.getUser(apiOp)
	if userPerms, ok := u.perms[user]; ok {
		if p, ok := userPerms[schema.ID]; ok {
			return p.read
		}
	}
	return false
}

func (u *userAccessControl) canWrite(apiOp *types.APIRequest, schema *types.APISchema) bool {
	user := u.getUser(apiOp)
	if userPerms, ok := u.perms[user]; ok {
		if p, ok := userPerms[schema.ID]; ok {
			return p.write
		}
	}
	return false
}

func (u *userAccessControl) CanCreate(apiOp *types.APIRequest, schema *types.APISchema) error {
	if u.canWrite(apiOp, schema) {
		return nil
	}
	return apierror.NewAPIError(validation.PermissionDenied, fmt.Sprintf("user %q cannot create %s", u.getUser(apiOp), schema.ID))
}

func (u *userAccessControl) CanGet(apiOp *types.APIRequest, schema *types.APISchema) error {
	if u.canRead(apiOp, schema) {
		return nil
	}
	return apierror.NewAPIError(validation.PermissionDenied, fmt.Sprintf("user %q cannot get %s", u.getUser(apiOp), schema.ID))
}

func (u *userAccessControl) CanList(apiOp *types.APIRequest, schema *types.APISchema) error {
	if u.canRead(apiOp, schema) {
		return nil
	}
	return apierror.NewAPIError(validation.PermissionDenied, fmt.Sprintf("user %q cannot list %s", u.getUser(apiOp), schema.ID))
}

func (u *userAccessControl) CanUpdate(apiOp *types.APIRequest, _ types.APIObject, schema *types.APISchema) error {
	if u.canWrite(apiOp, schema) {
		return nil
	}
	return apierror.NewAPIError(validation.PermissionDenied, fmt.Sprintf("user %q cannot update %s", u.getUser(apiOp), schema.ID))
}

func (u *userAccessControl) CanPatch(apiOp *types.APIRequest, _ types.APIObject, schema *types.APISchema) error {
	if u.canWrite(apiOp, schema) {
		return nil
	}
	return apierror.NewAPIError(validation.PermissionDenied, fmt.Sprintf("user %q cannot patch %s", u.getUser(apiOp), schema.ID))
}

func (u *userAccessControl) CanDelete(apiOp *types.APIRequest, _ types.APIObject, schema *types.APISchema) error {
	if u.canWrite(apiOp, schema) {
		return nil
	}
	return apierror.NewAPIError(validation.PermissionDenied, fmt.Sprintf("user %q cannot delete %s", u.getUser(apiOp), schema.ID))
}

func (u *userAccessControl) CanWatch(apiOp *types.APIRequest, schema *types.APISchema) error {
	return u.CanList(apiOp, schema)
}

func (u *userAccessControl) CanDo(apiOp *types.APIRequest, resource, verb, namespace, name string) error {
	schema := apiOp.Schemas.LookupSchema(resource)
	if schema == nil {
		return apierror.NewAPIError(validation.PermissionDenied, fmt.Sprintf("unknown resource %s", resource))
	}
	switch verb {
	case http.MethodGet:
		return u.CanList(apiOp, schema)
	case http.MethodPost:
		return u.CanCreate(apiOp, schema)
	case http.MethodPut:
		return u.CanUpdate(apiOp, types.APIObject{}, schema)
	case http.MethodPatch:
		return u.CanPatch(apiOp, types.APIObject{}, schema)
	case http.MethodDelete:
		return u.CanDelete(apiOp, types.APIObject{}, schema)
	default:
		return apierror.NewAPIError(validation.PermissionDenied, fmt.Sprintf("unknown verb %s", verb))
	}
}

func (u *userAccessControl) CanAction(apiOp *types.APIRequest, schema *types.APISchema, name string) error {
	if u.canWrite(apiOp, schema) {
		return nil
	}
	return apierror.NewAPIError(validation.PermissionDenied, fmt.Sprintf("user %q cannot perform action %s on %s", u.getUser(apiOp), name, schema.ID))
}

// Compile-time proof that userAccessControl satisfies types.AccessControl.
var _ types.AccessControl = (*userAccessControl)(nil)

// newUserAccessControl builds the access control table for all six test users.
//
// Permissions:
//   - alice: admin on dogs and cats  (read + write on both)
//   - bob:   admin on dogs, read-only on cats
//   - carol: admin on cats, read-only on dogs
//   - doug:  read-only on dogs, no access to cats
//   - enid:  read-only on cats, no access to dogs
//   - fred:  no access to any resource
func newUserAccessControl() types.AccessControl {
	rw := resourcePerm{read: true, write: true}
	ro := resourcePerm{read: true, write: false}

	return &userAccessControl{
		perms: map[string]map[string]resourcePerm{
			"alice": {"dog": rw, "cat": rw},
			"bob":   {"dog": rw, "cat": ro},
			"carol": {"dog": ro, "cat": rw},
			"doug":  {"dog": ro},
			"enid":  {"cat": ro},
			"fred":  {},
		},
	}
}

// doRequest sends an HTTP request with the given method/URL/body and an
// X-User header identifying the caller.
func doRequest(t *testing.T, method, url, user string, body []byte) *http.Response {
	t.Helper()
	var req *http.Request
	var err error
	if len(body) > 0 {
		req, err = http.NewRequest(method, url, bytes.NewReader(body))
	} else {
		req, err = http.NewRequest(method, url, nil)
	}
	require.NoError(t, err)
	req.Header.Set("X-User", user)
	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	return resp
}

// newAccessControlServer builds a fresh httptest.Server backed by independent
// DogStore and CatStore instances and the shared userAccessControl. Each call
// returns an isolated server so that write operations in one user's sub-test
// do not affect another user's sub-test.
func newAccessControlServer(t *testing.T) *httptest.Server {
	t.Helper()
	s := server.DefaultAPIServer()
	s.AccessControl = newUserAccessControl()
	s.Schemas.MustImportAndCustomize(Dog{}, func(schema *types.APISchema) {
		schema.Store = NewDogStore(nil)
		schema.CollectionMethods = []string{http.MethodGet, http.MethodPost}
		schema.ResourceMethods = []string{http.MethodGet, http.MethodPut, http.MethodPatch, http.MethodDelete}
	})
	s.Schemas.MustImportAndCustomize(Cat{}, func(schema *types.APISchema) {
		schema.Store = NewCatStore(nil)
		schema.CollectionMethods = []string{http.MethodGet, http.MethodPost}
		schema.ResourceMethods = []string{http.MethodGet, http.MethodPut, http.MethodPatch, http.MethodDelete}
	})
	ts := httptest.NewServer(newMultiPrefixRouter(s))
	t.Cleanup(ts.Close)
	return ts
}

// TestAccessControl_UsersOnDogsAndCats verifies that the custom
// userAccessControl correctly enforces per-user, per-resource permissions for
// six users across two resource types.
func TestAccessControl_UsersOnDogsAndCats(t *testing.T) {
	type opTest struct {
		name           string
		method         string
		path           string
		body           []byte
		wantStatusCode int
	}

	// Access matrix: for each user, define the expected HTTP status for each
	// operation on dogs and cats.  A zero status means "don't test this op".
	type userCase struct {
		user string
		ops  []opTest
	}

	cases := []userCase{
		{
			user: "alice",
			// Alice has admin access to both dogs and cats.
			ops: []opTest{
				{"list dogs", http.MethodGet, "/v1/dogs", nil, http.StatusOK},
				{"get dog by id", http.MethodGet, "/v1/dogs/pluto", nil, http.StatusOK},
				{"create dog", http.MethodPost, "/v1/dogs", []byte(`{"id":"fido","name":"mutt"}`), http.StatusCreated},
				{"update dog", http.MethodPut, "/v1/dogs/krypto", []byte(`{"name":"krypto-updated"}`), http.StatusOK},
				{"delete dog", http.MethodDelete, "/v1/dogs/pluto", nil, http.StatusOK},
				{"list cats", http.MethodGet, "/v1/cats", nil, http.StatusOK},
				{"get cat by id", http.MethodGet, "/v1/cats/felix", nil, http.StatusOK},
				{"create cat", http.MethodPost, "/v1/cats", []byte(`{"id":"whiskers","name":"tabby"}`), http.StatusCreated},
				{"update cat", http.MethodPut, "/v1/cats/fritz", []byte(`{"name":"fritz-updated"}`), http.StatusOK},
				{"delete cat", http.MethodDelete, "/v1/cats/boris", nil, http.StatusOK},
			},
		},
		{
			user: "bob",
			// Bob has admin on dogs but can only read cats.
			ops: []opTest{
				{"list dogs", http.MethodGet, "/v1/dogs", nil, http.StatusOK},
				{"get dog by id", http.MethodGet, "/v1/dogs/pluto", nil, http.StatusOK},
				{"create dog", http.MethodPost, "/v1/dogs", []byte(`{"id":"rex","name":"shepherd"}`), http.StatusCreated},
				{"update dog", http.MethodPut, "/v1/dogs/krypto", []byte(`{"name":"krypto-updated"}`), http.StatusOK},
				{"delete dog", http.MethodDelete, "/v1/dogs/pluto", nil, http.StatusOK},
				{"list cats", http.MethodGet, "/v1/cats", nil, http.StatusOK},
				{"get cat by id", http.MethodGet, "/v1/cats/felix", nil, http.StatusOK},
				{"cannot create cat", http.MethodPost, "/v1/cats", []byte(`{"id":"muffin","name":"persian"}`), http.StatusForbidden},
				{"cannot update cat", http.MethodPut, "/v1/cats/fritz", []byte(`{"name":"fritz-updated"}`), http.StatusForbidden},
				{"cannot delete cat", http.MethodDelete, "/v1/cats/boris", nil, http.StatusForbidden},
			},
		},
		{
			user: "carol",
			// Carol has admin on cats but can only read dogs.
			ops: []opTest{
				{"list dogs", http.MethodGet, "/v1/dogs", nil, http.StatusOK},
				{"get dog by id", http.MethodGet, "/v1/dogs/pluto", nil, http.StatusOK},
				{"cannot create dog", http.MethodPost, "/v1/dogs", []byte(`{"id":"buddy","name":"lab"}`), http.StatusForbidden},
				{"cannot update dog", http.MethodPut, "/v1/dogs/krypto", []byte(`{"name":"krypto-updated"}`), http.StatusForbidden},
				{"cannot delete dog", http.MethodDelete, "/v1/dogs/pluto", nil, http.StatusForbidden},
				{"list cats", http.MethodGet, "/v1/cats", nil, http.StatusOK},
				{"get cat by id", http.MethodGet, "/v1/cats/felix", nil, http.StatusOK},
				{"create cat", http.MethodPost, "/v1/cats", []byte(`{"id":"muffin","name":"persian"}`), http.StatusCreated},
				{"update cat", http.MethodPut, "/v1/cats/fritz", []byte(`{"name":"fritz-updated"}`), http.StatusOK},
				{"delete cat", http.MethodDelete, "/v1/cats/boris", nil, http.StatusOK},
			},
		},
		{
			user: "doug",
			// Doug can only read dogs; no access to cats.
			ops: []opTest{
				{"list dogs", http.MethodGet, "/v1/dogs", nil, http.StatusOK},
				{"get dog by id", http.MethodGet, "/v1/dogs/pluto", nil, http.StatusOK},
				{"cannot create dog", http.MethodPost, "/v1/dogs", []byte(`{"id":"buddy","name":"lab"}`), http.StatusForbidden},
				{"cannot update dog", http.MethodPut, "/v1/dogs/krypto", []byte(`{"name":"krypto-updated"}`), http.StatusForbidden},
				{"cannot delete dog", http.MethodDelete, "/v1/dogs/pluto", nil, http.StatusForbidden},
				{"cannot list cats", http.MethodGet, "/v1/cats", nil, http.StatusForbidden},
				{"cannot get cat by id", http.MethodGet, "/v1/cats/felix", nil, http.StatusForbidden},
				{"cannot create cat", http.MethodPost, "/v1/cats", []byte(`{"id":"muffin","name":"persian"}`), http.StatusForbidden},
				{"cannot update cat", http.MethodPut, "/v1/cats/fritz", []byte(`{"name":"fritz-updated"}`), http.StatusForbidden},
				{"cannot delete cat", http.MethodDelete, "/v1/cats/boris", nil, http.StatusForbidden},
			},
		},
		{
			user: "enid",
			// Enid can only read cats; no access to dogs.
			ops: []opTest{
				{"cannot list dogs", http.MethodGet, "/v1/dogs", nil, http.StatusForbidden},
				{"cannot get dog by id", http.MethodGet, "/v1/dogs/pluto", nil, http.StatusForbidden},
				{"cannot create dog", http.MethodPost, "/v1/dogs", []byte(`{"id":"buddy","name":"lab"}`), http.StatusForbidden},
				{"cannot update dog", http.MethodPut, "/v1/dogs/krypto", []byte(`{"name":"krypto-updated"}`), http.StatusForbidden},
				{"cannot delete dog", http.MethodDelete, "/v1/dogs/pluto", nil, http.StatusForbidden},
				{"list cats", http.MethodGet, "/v1/cats", nil, http.StatusOK},
				{"get cat by id", http.MethodGet, "/v1/cats/felix", nil, http.StatusOK},
				{"cannot create cat", http.MethodPost, "/v1/cats", []byte(`{"id":"muffin","name":"persian"}`), http.StatusForbidden},
				{"cannot update cat", http.MethodPut, "/v1/cats/fritz", []byte(`{"name":"fritz-updated"}`), http.StatusForbidden},
				{"cannot delete cat", http.MethodDelete, "/v1/cats/boris", nil, http.StatusForbidden},
			},
		},
		{
			user: "fred",
			// Fred has no access to any resource.
			ops: []opTest{
				{"cannot list dogs", http.MethodGet, "/v1/dogs", nil, http.StatusForbidden},
				{"cannot get dog by id", http.MethodGet, "/v1/dogs/pluto", nil, http.StatusForbidden},
				{"cannot create dog", http.MethodPost, "/v1/dogs", []byte(`{"id":"buddy","name":"lab"}`), http.StatusForbidden},
				{"cannot update dog", http.MethodPut, "/v1/dogs/krypto", []byte(`{"name":"krypto-updated"}`), http.StatusForbidden},
				{"cannot delete dog", http.MethodDelete, "/v1/dogs/pluto", nil, http.StatusForbidden},
				{"cannot list cats", http.MethodGet, "/v1/cats", nil, http.StatusForbidden},
				{"cannot get cat by id", http.MethodGet, "/v1/cats/felix", nil, http.StatusForbidden},
				{"cannot create cat", http.MethodPost, "/v1/cats", []byte(`{"id":"muffin","name":"persian"}`), http.StatusForbidden},
				{"cannot update cat", http.MethodPut, "/v1/cats/fritz", []byte(`{"name":"fritz-updated"}`), http.StatusForbidden},
				{"cannot delete cat", http.MethodDelete, "/v1/cats/boris", nil, http.StatusForbidden},
			},
		},
	}

	for _, uc := range cases {
		uc := uc
		t.Run(uc.user, func(t *testing.T) {
			// Fresh stores per user so write operations don't interfere.
			ts := newAccessControlServer(t)
			for _, op := range uc.ops {
				op := op
				t.Run(op.name, func(t *testing.T) {
					resp := doRequest(t, op.method, ts.URL+op.path, uc.user, op.body)
					defer resp.Body.Close()
					assert.Equal(t, op.wantStatusCode, resp.StatusCode)
				})
			}
		})
	}
}

Rancher API Server
==================

This repo contains the low level API server framework used to create REST APIs
consumed by Rancher projects such as
[github.com/rancher/ui](https://github.com/rancher/ui) and
[github.com/rancher/dashboard](https://github.com/rancher/dashboard). The
primary consumer of this framework is
[github.com/rancher/steve](https://github.com/rancher/steve).

Overview
--------

The API server is the interface between an HTTP client and a more complex
application like [rancher/steve](https://github.com/rancher/steve). The two
main components that are used to accomplish that are Schemas and Stores.

Schemas define metadata about an API type, describe CRUD handlers for the type,
define formatting transformations, and declare the Store that will be used to
transform and store the object.

Stores provide a common interface to perform CRUD operations on objects. The
implementation of the interface commonly involves either storing the data as a
field on the store object, forwarding it to another nested store, or calling
out to an external resource like Kubernetes.

Types
-----

There are a few main types to be aware of.

### APISchema

[APISchema](https://pkg.go.dev/github.com/rancher/apiserver/pkg/types#APISchema)
adds additional functionality on top of [wrangler's Schema
type](https://pkg.go.dev/github.com/rancher/wrangler/v2/pkg/schemas#Schema).
In addition to metadata about the type of object it represents, it also defines
CRUD handlers, formatting transformations, and the backing Store.

### Store

[Store](https://pkg.go.dev/github.com/rancher/apiserver/pkg/types#Store) is an
interface for interacting with `APIObject`s, `APIObjectList`s, and `APIEvent`s.

### APIRequest

[APIRequest](https://pkg.go.dev/github.com/rancher/apiserver/pkg/types#APIRequest)
is a parsed version of an `http.Request` that provides a standardized way of
interacting with a request. The default parser makes a set of assumptions about
how the request is formatted and routed so that it can populate fields such as
`Name`, `Namespace`, `Type`, or `Query`, among others. On top of the data found
in the request, `APIRequest` stores additional context that can be passed to
any function that needs to handle the request, such as the server's full set of
schemas, an access control interface, a response writer and error handler.

### APIObject

[APIObject](https://pkg.go.dev/github.com/rancher/apiserver/pkg/types#APIObject)
is a wrapper around an underlying object. The struct provides the object's type
and ID along with the unmodified object itself. If the underlying API object is
a Kubernetes resource, the ID is the object's name and namespace for namespaced
objects, or just its name for global objects. The type is the resource name and
API group. It also includes any warnings that may have been emitted while
processing the object.

### APIObjectList

[APIObjectList](https://pkg.go.dev/github.com/rancher/apiserver/pkg/types#APIObjectList)
is returned for list requests. It includes the slice of objects returned as
well as chunking and pagination metadata if the list is not complete.

### APIEvent

[APIEvent](https://pkg.go.dev/github.com/rancher/apiserver/pkg/types#APIEvent)
is emitted on a channel created for a watch request. It is a wrapper for a
Kubernetes event.

Usage
-----

The API server starts with an HTTP server:

```go
import "github.com/rancher/apiserver/pkg/server"
s := server.DefaultAPIServer()
```

Add schemas by defining a Go struct and importing an empty instance of it on to
the base schema list:

```go
type Duck struct{
    Name string `json:"name"`
}
s.Schemas.MustImportAndCustomize(Duck{}, nil)
```

If the API for this type needs to keep any state, a Store needs to be defined
in the customize function:

```go
import (
    "github.com/rancher/apiserver/pkg/types"
    "github.com/rancher/apiserver/pkg/store/empty"
)
type DuckStore struct {
    ducks map[string]Duck
}
func (d *DuckStore) ByID(apiOp *types.APIRequest, schema *types.APISchema, id string) (types.APIObject, error) {
    return types.APIObject{
        Type: "ducks",
        ID: id,
        Object: ducks[id],
    }, nil
}
// implement the rest of the Store interface
s.Schemas.MustImportAndCustomize(Duck{}, func(schema *types.APISchema) {
    schema.Store = &DuckStore{}
}
```

To make this an HTTP-accessible API, define allowed HTTP methods for a single
resource or for a collection:

```go
s.Schemas.MustImportAndCustomize(Duck{}, func(schema *types.APISchema) {
    schema.Store = &DuckStore{}
    schema.ResourceMethods: []string{"GET"},
    schema.CollectionMethods: []string{"GET"},
}
```

If HTTP methods are not defined on a schema, that schema can still be used in a
response, it just can't be queried or manipulated by a client. The
[error](#error) and [collection](#collection) built-in schemas are examples of
this kind of internal schema.

`MustImportAndCustomize` is a convenience wrapper around `MustAddSchema`, which
could also be used directly if desired:

```go
import "github.com/rancher/wrangler/v2/pkg/schemas"
s.Schemas.MustAddSchema(types.APISchema{
    Schema: &schemas.Schema{
        ID: "duck",
        ResourceFields: map[string]schemas.Field{
            "name": {Type: "string"},
        },
    },
    Store: &DuckStore{},
})
```

Routes need to be defined in order for requests to be routed to the correct
schema. The parser assumes that some or all of these variables may be defined
in the as part of a [gorilla/mux](https://pkg.go.dev/github.com/gorilla/mux)
router: "type", "name", "namespace", "link", "prefix", "action". It uses these
assumptions to decode the `http.Request` into an `APIRequest`. For example,
for a route like:

```go
import "github.com/gorilla/mux"
router := mux.NewRouter()
router.Handle("/{prefix}/{type}/{namespace}/{name}", s)
```

then a request like

```
GET /pond/duck/mallard/bob
```

would generate an APIRequest like

```go
APIRequest{
    Type: "duck",
    Prefix: "pond",
    Namespace: "mallard",
    Name: "bob",
    Method: "GET",
}
```

and route the request to the "duck" registered schema.

An example server can be found in [example.go](./example.go) and run on port
8080 with

```sh
go run example.go
```

Built-ins
---------

API server provides a set of built-in and convenience schemas:

### schema

Provides read-only access to any schema definition.

### error

Defines the format for an error response.

### collection

Defines the format for a list of objects.

### apiroot

Not built in to the default schemas, but can be added with:

```go
import "github.com/rancher/apiserver/pkg/store/apiroot"
apiroot.Register(s.Schemas, []string{"v1"})
```

This adds one or more "roots" relative to which schemas are defined, to allow
for more than one schema version to coexist.

### subscribe

Also not built in, but can be added with

```go
import "github.com/rancher/apiserver/pkg/subscribe"
subscribe.Register(s.Schemas, nil, "")
```

The `Subscribe` schema provides special handling for listening for events on a
channel and passing them through a websocket.

A useful tool for connecting to a websocket without a browser is
[websocat](https://github.com/vi/websocat).

A subscription stream is started by making a websocket request for the
`subscribe` type, which is routed to the
[Subscribe](https://pkg.go.dev/github.com/rancher/apiserver/pkg/subscribe#Subscribe)
schema. It uses a custom handler to upgrade the connection to a websocket
connection.

The event stream is started when the client requests a resource type over the
websocket connection. The message from the client consists of the resource type
and optional filtering parameters. For example:

```
{"resourceType": "apps.deployments", "namespace": "default", "resourceVersion": "1000"}
```

will start watching events for the "apps.deployments" collection in namespace
"default" starting with the collection resource version "1000" (see [the
Kubernetes documentation](https://kubernetes.io/docs/reference/using-api/api-concepts/#resource-versions)
for a detailed discussion of resource version semantics). Under the hood, the
API server calls the `Watch` method for the schema's store for the resource
type.

The watch could be started for an individual resource by specifying the "id"
field, for a set of labeled resources by using the "selector" field, or for all
resources by omitting the "namespace" field.

To stop a watch deliberately, issue a "stop" message:

```
{"stop": true, "resourceType": "apps.deployments"}
```

Otherwise, the connection will time out after 30 minutes, and will terminate
with a message with name "resource.stop". The client is responsible for
restarting the connection.

If an error is encounted, a message with name "resource.error" will be sent
with error details in the message.

Access Control
--------------

Access control is defined on the server. By default, access control is based on
the defined `ResourceMethods` and `CollectionMethods` on the `Schema`, and the
access is the same for every request. More complex access control, using RBAC,
for instance, can be defined by overriding the `SchemaBasedAccess` struct:

```go
import (
    "k8s.io/apiserver/pkg/endpoints/request"
    "github.com/rancher/apiserver/pkg/apierror"
    "github.com/rancher/apiserver/pkg/server"
    "github.com/rancher/apiserver/pkg/types"
)
type accessControl struct{
    server.SchemaBasedAccess
}
func (a *accessControl) CanList(apiOp *types.APIRequest, schema *types.APISchema) error {
    user, ok := request.UserFrom(apiOp.Context())
    if ok && user.GetName() == "george" {
        return apierror.NewAPIError(validation.PermissionDenied, "no Georges allowed")
    }
    return nil
}
s.AccessControl = &accessControl{}
```

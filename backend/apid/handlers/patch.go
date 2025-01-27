package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/gorilla/mux"
	corev2 "github.com/sensu/core/v2"
	corev3 "github.com/sensu/core/v3"
	"github.com/sensu/sensu-go/backend/apid/actions"
	"github.com/sensu/sensu-go/backend/store"
	"github.com/sensu/sensu-go/backend/store/patch"
	storev2 "github.com/sensu/sensu-go/backend/store/v2"
)

const (
	mergePatchContentType = "application/merge-patch+json"
	jsonPatchContentType  = "application/json-patch+json"

	ifMatchHeader     = "If-Match"
	ifNoneMatchHeader = "If-None-Match"
)

// acceptedContentTypes contains the list of content types we accept
var acceptedContentTypes = []string{mergePatchContentType}

// PatchResource patches a given resource, using the request body as the patch
func (h Handlers[R, T]) PatchResource(r *http.Request) (HandlerResponse, error) {
	var response HandlerResponse

	// Read the request body
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return response, actions.NewError(
			actions.InvalidArgument,
			fmt.Errorf("could not read the request body: %s", err),
		)
	}

	var patcher patch.Patcher

	// Determine the requested PATCH operation based on the Content-Type header
	// and initialize a patcher
	switch contentType := r.Header.Get("Content-Type"); contentType {
	case mergePatchContentType, "": // Use merge patch as fallback value
		patcher = &patch.Merge{MergePatch: body}
	case jsonPatchContentType:
		return response, actions.NewError(
			actions.InvalidArgument,
			fmt.Errorf("JSON Patch is not supported yet. Allowed values: %s", strings.Join(acceptedContentTypes, ", ")),
		)
	default:
		return response, actions.NewError(
			actions.InvalidArgument,
			fmt.Errorf("invalid Content-Type header: %s.  Allowed values: %s", contentType, strings.Join(acceptedContentTypes, ", ")),
		)
	}

	ctx := r.Context()

	// Determine if we have a conditional request
	if value := r.Header.Get(ifMatchHeader); value != "" {
		ifMatch, err := storev2.ReadIfMatch(value)
		if err != nil {
			return response, actions.NewError(actions.InvalidArgument, fmt.Errorf("invalid If-Match header: %s", err))
		}
		ctx = storev2.ContextWithIfMatch(ctx, ifMatch)
	}
	if value := r.Header.Get(ifNoneMatchHeader); value != "" {
		ifNoneMatch, err := storev2.ReadIfNoneMatch(value)
		if err != nil {
			return response, actions.NewError(actions.InvalidArgument, fmt.Errorf("invalid If-None-Match header: %s", err))
		}
		ctx = storev2.ContextWithIfNoneMatch(ctx, ifNoneMatch)
	}

	// Retrieve the name & namespace of the resource via the route variables
	params := mux.Vars(r)
	name, err := url.PathUnescape(params["id"])
	if err != nil {
		return response, err
	}
	namespace, err := url.PathUnescape(params["namespace"])
	if err != nil {
		return response, err
	}

	// Validate that the patch does not alter the namespace nor the name
	if err := validatePatch(body, params); err != nil {
		return response, actions.NewError(actions.InvalidArgument, err)
	}

	resource, err := h.patchV3Resource(ctx, name, namespace, patcher)
	response.Resource = resource
	return response, err
}

func (h Handlers[R, T]) patchV3Resource(ctx context.Context, name, namespace string, patcher patch.Patcher) (corev3.Resource, error) {
	gstore := storev2.Of[R](h.Store)

	id := storev2.ID{Namespace: namespace, Name: name}
	if err := gstore.Patch(ctx, id, patcher); err != nil {
		switch err := err.(type) {
		case *store.ErrNotFound:
			return nil, actions.NewError(actions.NotFound, err)
		case *store.ErrNotValid:
			return nil, actions.NewError(actions.InvalidArgument, err)
		case *store.ErrPreconditionFailed:
			return nil, actions.NewError(actions.PreconditionFailed, err)
		default:
			return nil, actions.NewError(actions.InternalErr, err)
		}
	}

	return nil, nil
}

func validatePatch(data []byte, vars map[string]string) error {
	type body struct {
		Metadata *corev2.ObjectMeta `json:"metadata"`
	}

	b := &body{}

	if err := json.Unmarshal(data, b); err != nil {
		return err
	}

	if b.Metadata == nil {
		return nil
	}

	namespace, err := url.PathUnescape(vars["namespace"])
	if err != nil {
		return err
	}
	if b.Metadata.Namespace != "" && b.Metadata.Namespace != namespace {
		return fmt.Errorf(
			"the namespace of the resource (%s) does not match the namespace in the URI (%s)",
			b.Metadata.Namespace,
			namespace,
		)
	}

	name, err := url.PathUnescape(vars["id"])
	if err != nil {
		return err
	}
	if b.Metadata.Name != "" && b.Metadata.Name != name {
		return fmt.Errorf(
			"the name of the resource (%s) does not match the name in the URI (%s)",
			b.Metadata.Name,
			name,
		)
	}

	return nil
}

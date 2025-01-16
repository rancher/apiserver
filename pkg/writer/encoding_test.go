package writer

import (
	"github.com/golang/mock/gomock"
	"net/http"
	"testing"

	"github.com/rancher/apiserver/pkg/fakes"
	"github.com/rancher/apiserver/pkg/types"
	"github.com/stretchr/testify/assert"
)

func TestAddLinks(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockURLBuilder := fakes.NewMockURLBuilder(ctrl)
	mockAccessControl := fakes.NewMockAccessControl(ctrl)

	schema := &types.APISchema{
		LinkHandlers:   map[string]http.Handler{"customLink": nil},
		ActionHandlers: map[string]http.Handler{"customAction": nil},
	}
	context := &types.APIRequest{
		URLBuilder:    mockURLBuilder,
		AccessControl: mockAccessControl,
	}
	input := types.APIObject{Type: "testType", ID: "testID"}
	rawResource := &types.RawResource{
		ID:     "testResourceID",
		Schema: schema,
		Links:  map[string]string{},
	}

	mockURLBuilder.EXPECT().ResourceLink(schema, "testResourceID").Return("selfLink").AnyTimes()
	mockURLBuilder.EXPECT().Link(schema, "testResourceID", "customLink").Return("customLinkURL").AnyTimes()
	mockURLBuilder.EXPECT().Action(schema, "testResourceID", "customAction").Return("customActionURL").AnyTimes()

	mockAccessControl.EXPECT().CanUpdate(context, input, schema).Return(nil).AnyTimes()
	mockAccessControl.EXPECT().CanDelete(context, input, schema).Return(nil).AnyTimes()
	mockAccessControl.EXPECT().CanPatch(context, input, schema).Return(nil).AnyTimes()

	writer := &EncodingResponseWriter{}
	writer.addLinks(schema, context, input, rawResource)

	assert.Equal(t, "selfLink", rawResource.Links["self"])
	assert.Equal(t, "selfLink", rawResource.Links["update"])
	assert.Equal(t, "selfLink", rawResource.Links["remove"])
	assert.Equal(t, "selfLink", rawResource.Links["patch"])
	assert.Equal(t, "customLinkURL", rawResource.Links["customLink"])
	assert.Equal(t, "customActionURL", rawResource.Actions["customAction"])
}

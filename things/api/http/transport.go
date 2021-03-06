package http

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"

	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/go-zoo/bone"
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/things"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const contentType = "application/json"

var (
	errUnsupportedContentType = errors.New("unsupported content type")
	errInvalidQueryParams     = errors.New("invalid query params")
)

// MakeHandler returns a HTTP handler for API endpoints.
func MakeHandler(svc things.Service) http.Handler {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(encodeError),
	}

	r := bone.New()

	r.Post("/things", kithttp.NewServer(
		addThingEndpoint(svc),
		decodeThingCreation,
		encodeResponse,
		opts...,
	))

	r.Put("/things/:id", kithttp.NewServer(
		updateThingEndpoint(svc),
		decodeThingUpdate,
		encodeResponse,
		opts...,
	))

	r.Delete("/things/:id", kithttp.NewServer(
		removeThingEndpoint(svc),
		decodeView,
		encodeResponse,
		opts...,
	))

	r.Get("/things/:id", kithttp.NewServer(
		viewThingEndpoint(svc),
		decodeView,
		encodeResponse,
		opts...,
	))

	r.Get("/things", kithttp.NewServer(
		listThingsEndpoint(svc),
		decodeList,
		encodeResponse,
		opts...,
	))

	r.Post("/channels", kithttp.NewServer(
		createChannelEndpoint(svc),
		decodeChannelCreation,
		encodeResponse,
		opts...,
	))

	r.Put("/channels/:id", kithttp.NewServer(
		updateChannelEndpoint(svc),
		decodeChannelUpdate,
		encodeResponse,
		opts...,
	))

	r.Delete("/channels/:id", kithttp.NewServer(
		removeChannelEndpoint(svc),
		decodeView,
		encodeResponse,
		opts...,
	))

	r.Get("/channels/:id", kithttp.NewServer(
		viewChannelEndpoint(svc),
		decodeView,
		encodeResponse,
		opts...,
	))

	r.Get("/channels", kithttp.NewServer(
		listChannelsEndpoint(svc),
		decodeList,
		encodeResponse,
		opts...,
	))

	r.Put("/channels/:chanId/things/:thingId", kithttp.NewServer(
		connectEndpoint(svc),
		decodeConnection,
		encodeResponse,
		opts...,
	))

	r.Delete("/channels/:chanId/things/:thingId", kithttp.NewServer(
		disconnectEndpoint(svc),
		decodeConnection,
		encodeResponse,
		opts...,
	))

	r.GetFunc("/version", mainflux.Version("things"))
	r.Handle("/metrics", promhttp.Handler())

	return r
}

func decodeThingCreation(_ context.Context, r *http.Request) (interface{}, error) {
	if r.Header.Get("Content-Type") != contentType {
		return nil, errUnsupportedContentType
	}

	var thing things.Thing
	if err := json.NewDecoder(r.Body).Decode(&thing); err != nil {
		return nil, err
	}

	req := addThingReq{
		key:   r.Header.Get("Authorization"),
		thing: thing,
	}

	return req, nil
}

func decodeThingUpdate(_ context.Context, r *http.Request) (interface{}, error) {
	if r.Header.Get("Content-Type") != contentType {
		return nil, errUnsupportedContentType
	}

	var thing things.Thing
	if err := json.NewDecoder(r.Body).Decode(&thing); err != nil {
		return nil, err
	}

	req := updateThingReq{
		key:   r.Header.Get("Authorization"),
		id:    bone.GetValue(r, "id"),
		thing: thing,
	}

	return req, nil
}

func decodeChannelCreation(_ context.Context, r *http.Request) (interface{}, error) {
	if r.Header.Get("Content-Type") != contentType {
		return nil, errUnsupportedContentType
	}

	var channel things.Channel
	if err := json.NewDecoder(r.Body).Decode(&channel); err != nil {
		return nil, err
	}

	req := createChannelReq{
		key:     r.Header.Get("Authorization"),
		channel: channel,
	}

	return req, nil
}

func decodeChannelUpdate(_ context.Context, r *http.Request) (interface{}, error) {
	if r.Header.Get("Content-Type") != contentType {
		return nil, errUnsupportedContentType
	}

	var channel things.Channel
	if err := json.NewDecoder(r.Body).Decode(&channel); err != nil {
		return nil, err
	}

	req := updateChannelReq{
		key:     r.Header.Get("Authorization"),
		id:      bone.GetValue(r, "id"),
		channel: channel,
	}

	return req, nil
}

func decodeView(_ context.Context, r *http.Request) (interface{}, error) {
	req := viewResourceReq{
		key: r.Header.Get("Authorization"),
		id:  bone.GetValue(r, "id"),
	}

	return req, nil
}

func decodeList(_ context.Context, r *http.Request) (interface{}, error) {
	q, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		return nil, errInvalidQueryParams
	}
	offset := 0
	limit := 10

	off, lmt := q["offset"], q["limit"]

	if len(off) > 1 || len(lmt) > 1 {
		return nil, errInvalidQueryParams
	}

	if len(off) == 1 {
		offset, err = strconv.Atoi(off[0])
		if err != nil {
			return nil, errInvalidQueryParams
		}
	}

	if len(lmt) == 1 {
		limit, err = strconv.Atoi(lmt[0])
		if err != nil {
			return nil, errInvalidQueryParams
		}
	}
	req := listResourcesReq{
		key:    r.Header.Get("Authorization"),
		offset: offset,
		limit:  limit,
	}

	return req, nil
}

func decodeConnection(_ context.Context, r *http.Request) (interface{}, error) {
	req := connectionReq{
		key:     r.Header.Get("Authorization"),
		chanID:  bone.GetValue(r, "chanId"),
		thingID: bone.GetValue(r, "thingId"),
	}

	return req, nil
}

func encodeResponse(_ context.Context, w http.ResponseWriter, response interface{}) error {
	w.Header().Set("Content-Type", contentType)

	if ar, ok := response.(mainflux.Response); ok {
		for k, v := range ar.Headers() {
			w.Header().Set(k, v)
		}

		w.WriteHeader(ar.Code())

		if ar.Empty() {
			return nil
		}
	}

	return json.NewEncoder(w).Encode(response)
}

func encodeError(_ context.Context, err error, w http.ResponseWriter) {
	w.Header().Set("Content-Type", contentType)

	switch err {
	case things.ErrMalformedEntity:
		w.WriteHeader(http.StatusBadRequest)
	case things.ErrUnauthorizedAccess:
		w.WriteHeader(http.StatusForbidden)
	case things.ErrNotFound:
		w.WriteHeader(http.StatusNotFound)
	case things.ErrConflict:
		w.WriteHeader(http.StatusConflict)
	case errUnsupportedContentType:
		w.WriteHeader(http.StatusUnsupportedMediaType)
	case errInvalidQueryParams:
		w.WriteHeader(http.StatusBadRequest)
	case io.ErrUnexpectedEOF:
		w.WriteHeader(http.StatusBadRequest)
	case io.EOF:
		w.WriteHeader(http.StatusBadRequest)
	default:
		switch err.(type) {
		case *json.SyntaxError:
			w.WriteHeader(http.StatusBadRequest)
		case *json.UnmarshalTypeError:
			w.WriteHeader(http.StatusBadRequest)
		default:
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
}

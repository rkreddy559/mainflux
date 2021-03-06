package things

import (
	"context"
	"errors"
	"time"

	"github.com/mainflux/mainflux"
)

var (
	// ErrConflict indicates usage of the existing email during account
	// registration.
	ErrConflict = errors.New("email already taken")

	// ErrMalformedEntity indicates malformed entity specification (e.g.
	// invalid username or password).
	ErrMalformedEntity = errors.New("malformed entity specification")

	// ErrUnauthorizedAccess indicates missing or invalid credentials provided
	// when accessing a protected resource.
	ErrUnauthorizedAccess = errors.New("missing or invalid credentials provided")

	// ErrNotFound indicates a non-existent entity request.
	ErrNotFound = errors.New("non-existent entity")
)

// Service specifies an API that must be fullfiled by the domain service
// implementation, and all of its decorators (e.g. logging & metrics).
type Service interface {
	// AddThing adds new thing to the user identified by the provided key.
	AddThing(string, Thing) (Thing, error)

	// UpdateThing updates the thing identified by the provided ID, that
	// belongs to the user identified by the provided key.
	UpdateThing(string, Thing) error

	// ViewThing retrieves data about the thing identified with the provided
	// ID, that belongs to the user identified by the provided key.
	ViewThing(string, string) (Thing, error)

	// ListThings retrieves data about subset of things that belongs to the
	// user identified by the provided key.
	ListThings(string, int, int) ([]Thing, error)

	// RemoveThing removes the thing identified with the provided ID, that
	// belongs to the user identified by the provided key.
	RemoveThing(string, string) error

	// CreateChannel adds new channel to the user identified by the provided key.
	CreateChannel(string, Channel) (Channel, error)

	// UpdateChannel updates the channel identified by the provided ID, that
	// belongs to the user identified by the provided key.
	UpdateChannel(string, Channel) error

	// ViewChannel retrieves data about the channel identified by the provided
	// ID, that belongs to the user identified by the provided key.
	ViewChannel(string, string) (Channel, error)

	// ListChannels retrieves data about subset of channels that belongs to the
	// user identified by the provided key.
	ListChannels(string, int, int) ([]Channel, error)

	// RemoveChannel removes the thing identified by the provided ID, that
	// belongs to the user identified by the provided key.
	RemoveChannel(string, string) error

	// Connect adds thing to the channel's list of connected things.
	Connect(string, string, string) error

	// Disconnect removes thing from the channel's list of connected
	// things.
	Disconnect(string, string, string) error

	// CanAccess determines whether the channel can be accessed using the
	// provided key and returns thing's id if access is allowed.
	CanAccess(string, string) (string, error)
}

var _ Service = (*thingsService)(nil)

type thingsService struct {
	users    mainflux.UsersServiceClient
	things   ThingRepository
	channels ChannelRepository
	idp      IdentityProvider
}

// New instantiates the things service implementation.
func New(users mainflux.UsersServiceClient, things ThingRepository, channels ChannelRepository, idp IdentityProvider) Service {
	return &thingsService{
		users:    users,
		things:   things,
		channels: channels,
		idp:      idp,
	}
}

func (ts *thingsService) AddThing(key string, thing Thing) (Thing, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	res, err := ts.users.Identify(ctx, &mainflux.Token{Value: key})
	if err != nil {
		return Thing{}, ErrUnauthorizedAccess
	}

	// TODO: drop completely in a separate ticket
	thing.ID = ts.idp.ID()
	thing.Owner = res.GetValue()
	thing.Key = ts.idp.ID()

	if _, err := ts.things.Save(thing); err != nil {
		return Thing{}, err
	}

	return thing, nil
}

func (ts *thingsService) UpdateThing(key string, thing Thing) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	res, err := ts.users.Identify(ctx, &mainflux.Token{Value: key})
	if err != nil {
		return ErrUnauthorizedAccess
	}

	thing.Owner = res.GetValue()

	return ts.things.Update(thing)
}

func (ts *thingsService) ViewThing(key, id string) (Thing, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	res, err := ts.users.Identify(ctx, &mainflux.Token{Value: key})
	if err != nil {
		return Thing{}, ErrUnauthorizedAccess
	}

	return ts.things.One(res.GetValue(), id)
}

func (ts *thingsService) ListThings(key string, offset, limit int) ([]Thing, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	res, err := ts.users.Identify(ctx, &mainflux.Token{Value: key})
	if err != nil {
		return nil, ErrUnauthorizedAccess
	}

	return ts.things.All(res.GetValue(), offset, limit), nil
}

func (ts *thingsService) RemoveThing(key, id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	res, err := ts.users.Identify(ctx, &mainflux.Token{Value: key})
	if err != nil {
		return ErrUnauthorizedAccess
	}

	return ts.things.Remove(res.GetValue(), id)
}

func (ts *thingsService) CreateChannel(key string, channel Channel) (Channel, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	res, err := ts.users.Identify(ctx, &mainflux.Token{Value: key})
	if err != nil {
		return Channel{}, ErrUnauthorizedAccess
	}

	// TODO: drop completely in a separate ticket
	channel.ID = ts.idp.ID()
	channel.Owner = res.GetValue()

	if _, err := ts.channels.Save(channel); err != nil {
		return Channel{}, err
	}

	return channel, nil
}

func (ts *thingsService) UpdateChannel(key string, channel Channel) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	res, err := ts.users.Identify(ctx, &mainflux.Token{Value: key})
	if err != nil {
		return ErrUnauthorizedAccess
	}

	channel.Owner = res.GetValue()
	return ts.channels.Update(channel)
}

func (ts *thingsService) ViewChannel(key, id string) (Channel, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	res, err := ts.users.Identify(ctx, &mainflux.Token{Value: key})
	if err != nil {
		return Channel{}, ErrUnauthorizedAccess
	}

	return ts.channels.One(res.GetValue(), id)
}

func (ts *thingsService) ListChannels(key string, offset, limit int) ([]Channel, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	res, err := ts.users.Identify(ctx, &mainflux.Token{Value: key})
	if err != nil {
		return nil, ErrUnauthorizedAccess
	}

	return ts.channels.All(res.GetValue(), offset, limit), nil
}

func (ts *thingsService) RemoveChannel(key, id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	res, err := ts.users.Identify(ctx, &mainflux.Token{Value: key})
	if err != nil {
		return ErrUnauthorizedAccess
	}

	return ts.channels.Remove(res.GetValue(), id)
}

func (ts *thingsService) Connect(key, chanID, thingID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	res, err := ts.users.Identify(ctx, &mainflux.Token{Value: key})
	if err != nil {
		return ErrUnauthorizedAccess
	}

	return ts.channels.Connect(res.GetValue(), chanID, thingID)
}

func (ts *thingsService) Disconnect(key, chanID, thingID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	res, err := ts.users.Identify(ctx, &mainflux.Token{Value: key})
	if err != nil {
		return ErrUnauthorizedAccess
	}

	return ts.channels.Disconnect(res.GetValue(), chanID, thingID)
}

func (ts *thingsService) CanAccess(key, channel string) (string, error) {
	thingID, err := ts.channels.HasThing(channel, key)
	if err != nil {
		return "", ErrUnauthorizedAccess
	}

	return thingID, nil
}
